import argparse
import json
import os
import sys


def fail(message):
    print(json.dumps({"ok": False, "error": message}), flush=True)
    return 1


def truthy(value):
    return str(value).strip().lower() in {"1", "true", "yes", "on", "enabled"}


def local_files_only():
    return truthy(os.getenv("FIREQUERY_LOCAL_FILES_ONLY", "0"))


def select_device():
    import torch

    if truthy(os.getenv("FIREQUERY_ENABLE_CUDA", "0")) and torch.cuda.is_available():
        return "cuda"
    return "cpu"


def mean_pool(last_hidden_state, attention_mask):
    import torch

    mask = attention_mask.unsqueeze(-1).expand(last_hidden_state.size()).float()
    pooled = (last_hidden_state * mask).sum(1) / mask.sum(1).clamp(min=1e-9)
    return torch.nn.functional.normalize(pooled, p=2, dim=1)


LABEL_DESCRIPTIONS = {
    "remember_information": "Store new durable information in memory.",
    "recall_information": "Retrieve known information from memory.",
    "build_context": "Build context to answer a task or question.",
    "explain_decision": "Explain why a memory decision or result happened.",
    "sync_memory": "Synchronize and enrich pending memory.",
    "relate_memory": "Create relations between memory items.",
    "forget_memory": "Delete or forget a memory item.",
    "consolidate_memory": "Merge, reinforce, or consolidate memory items.",
    "do_nothing": "Do nothing and avoid any memory operation.",
    "query_memory": "Read, search, or inspect memory without writing.",
    "suggest_write": "Suggest creating or updating memory through a write operation.",
    "request_confirmation": "Ask for explicit confirmation before a risky operation.",
}

ENTITY_LABELS = [
    "person",
    "organization",
    "product",
    "technology",
    "version",
    "issue",
    "document",
    "date",
    "location",
]


class State:
    def __init__(self):
        self._encoders = {}
        self._gliners = {}
        self.device = None

    def ensure_imports(self):
        import torch  # noqa: F401
        import transformers  # noqa: F401
        import sentencepiece  # noqa: F401
        import safetensors  # noqa: F401
        import gliner  # noqa: F401

    def get_device(self):
        if self.device is None:
            self.device = select_device()
        return self.device

    def load_encoder(self, model_id):
        if model_id not in self._encoders:
            from transformers import AutoModel, AutoTokenizer

            tokenizer = AutoTokenizer.from_pretrained(model_id, local_files_only=local_files_only())
            model = AutoModel.from_pretrained(model_id, local_files_only=local_files_only())
            model.eval()
            model.to(self.get_device())
            self._encoders[model_id] = (tokenizer, model)
        return self._encoders[model_id]

    def load_gliner(self, model_id):
        if model_id not in self._gliners:
            from gliner import GLiNER

            model = GLiNER.from_pretrained(model_id, local_files_only=local_files_only())
            self._gliners[model_id] = model
        return self._gliners[model_id]

    def encoder_dimension(self, model_id):
        _, model = self.load_encoder(model_id)
        return int(getattr(model.config, "hidden_size", 0) or 0)

    def embed(self, model_id, texts):
        import torch

        tokenizer, model = self.load_encoder(model_id)
        encoded = tokenizer(
            texts,
            padding=True,
            truncation=True,
            return_tensors="pt",
        )
        encoded = {key: value.to(self.get_device()) for key, value in encoded.items()}
        with torch.no_grad():
            outputs = model(**encoded)
        pooled = mean_pool(outputs.last_hidden_state, encoded["attention_mask"])
        return pooled.cpu().tolist()

    def classify(self, model_id, text, labels):
        label_texts = []
        for label in labels:
            label_texts.append(LABEL_DESCRIPTIONS.get(label, label.replace("_", " ")))

        input_vector = self.embed(model_id, [text])[0]
        label_vectors = self.embed(model_id, label_texts)

        scored = []
        for label, vector in zip(labels, label_vectors):
            score = cosine(input_vector, vector)
            scored.append({"label": label, "score": score})

        scored.sort(key=lambda item: (-item["score"], item["label"]))
        return scored

    def extract_entities(self, model_id, text):
        model = self.load_gliner(model_id)
        predictions = model.predict_entities(text, ENTITY_LABELS)
        entities = []
        for prediction in predictions:
            entities.append({
                "text": prediction.get("text", ""),
                "type": prediction.get("label", ""),
                "score": float(prediction.get("score", 0.0)),
            })
        entities.sort(key=lambda item: (-item["score"], item["text"], item["type"]))
        return entities


def cosine(left, right):
    numerator = 0.0
    left_norm = 0.0
    right_norm = 0.0
    for lval, rval in zip(left, right):
        numerator += float(lval) * float(rval)
        left_norm += float(lval) * float(lval)
        right_norm += float(rval) * float(rval)
    if left_norm == 0.0 or right_norm == 0.0:
        return 0.0
    return numerator / ((left_norm ** 0.5) * (right_norm ** 0.5))


def handle_request(state, request):
    op = request.get("op", "")
    model_id = request.get("model_id", "")

    if op == "health":
        state.ensure_imports()
        return {"ok": True, "device": state.get_device()}

    if op == "model_info":
        task = request.get("task", "")
        if task in {"embedding", "classification"}:
            return {"ok": True, "dimension": state.encoder_dimension(model_id)}
        if task == "entity_extraction":
            state.load_gliner(model_id)
            return {"ok": True, "dimension": 0}
        return {"ok": False, "error": f"unknown task: {task}"}

    if op == "classify":
        text = request.get("text", "")
        labels = request.get("labels", [])
        return {"ok": True, "labels": state.classify(model_id, text, labels)}

    if op == "extract_entities":
        text = request.get("text", "")
        return {"ok": True, "entities": state.extract_entities(model_id, text)}

    if op == "embed":
        texts = request.get("texts", [])
        return {"ok": True, "vectors": state.embed(model_id, texts)}

    return {"ok": False, "error": f"unknown operation: {op}"}


def run_healthcheck():
    try:
        state = State()
        state.ensure_imports()
        state.get_device()
        print(json.dumps({"ok": True}), flush=True)
        return 0
    except Exception as exc:
        return fail(str(exc))


def run_stdio():
    state = State()
    for line in sys.stdin:
        line = line.strip()
        if not line:
            continue
        try:
            request = json.loads(line)
            response = handle_request(state, request)
        except Exception as exc:
            response = {"ok": False, "error": str(exc)}
        print(json.dumps(response), flush=True)
    return 0


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--healthcheck", action="store_true")
    args = parser.parse_args()

    if args.healthcheck:
        return run_healthcheck()
    return run_stdio()


if __name__ == "__main__":
    raise SystemExit(main())
