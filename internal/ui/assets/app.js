'use strict';

const NODE_COLORS = {
  note:    '#94a3b8',
  fact:    '#4a9eff',
  event:   '#22c55e',
  concept: '#9b5cf6',
};

const EDGE_COLORS = {
  references: '#64748b',
  conflict:   '#ef4444',
  reinforce:  '#22c55e',
  complement: '#f59e0b',
  update:     '#3b82f6',
  duplicate:  '#6b7280',
  associated: '#a3a3a3',
};

function edgeColor(type) {
  return EDGE_COLORS[type] || '#64748b';
}

function nodeColor(kind) {
  return NODE_COLORS[kind] || NODE_COLORS.note;
}

let cy;
let nodeIndex = {};  // id → node data (from /api/graph)

async function init() {
  const resp = await fetch('/api/graph');
  if (!resp.ok) { showError('Failed to load graph'); return; }
  const data = await resp.json();

  nodeIndex = {};
  (data.nodes || []).forEach(n => { nodeIndex[n.id] = n; });

  const elements = [];

  (data.nodes || []).forEach(n => {
    elements.push({
      group: 'nodes',
      data: {
        id: n.id,
        label: truncate(n.label, 40),
        kind: n.kind || 'note',
      },
    });
  });

  (data.edges || []).forEach(e => {
    elements.push({
      group: 'edges',
      data: {
        id: e.id,
        source: e.from_id,
        target: e.to_id,
        type: e.type || 'associated',
        weight: e.weight || 1,
      },
    });
  });

  cy = cytoscape({
    container: document.getElementById('cy'),
    elements,
    style: cytoscapeStyle(),
    layout: { name: 'cose', animate: false, randomize: true, nodeRepulsion: 8000, idealEdgeLength: 100 },
    minZoom: 0.05,
    maxZoom: 5,
  });

  cy.on('tap', 'node', e => selectNode(e.target.id()));
  cy.on('tap', e => { if (e.target === cy) clearPanel(); });

  document.getElementById('btn-fit').addEventListener('click', () => cy.fit(undefined, 40));
  document.getElementById('btn-layout').addEventListener('click', rerunLayout);

  const n = (data.nodes || []).length;
  const ed = (data.edges || []).length;
  document.getElementById('stats').textContent = `${n} nodes · ${ed} edges`;
}

function rerunLayout() {
  if (!cy) return;
  cy.layout({ name: 'cose', animate: true, randomize: false, nodeRepulsion: 8000, idealEdgeLength: 100 }).run();
}

async function selectNode(id) {
  cy.elements().removeClass('highlighted dimmed');
  const node = cy.getElementById(id);
  const connected = node.closedNeighborhood();
  connected.addClass('highlighted');
  cy.elements().not(connected).addClass('dimmed');

  try {
    const resp = await fetch(`/api/node/${encodeURIComponent(id)}`);
    if (!resp.ok) throw new Error(`HTTP ${resp.status}`);
    const detail = await resp.json();
    renderPanel(detail);
  } catch (err) {
    showError(err.message);
  }
}

function clearPanel() {
  if (cy) cy.elements().removeClass('highlighted dimmed');
  document.getElementById('panel-placeholder').hidden = false;
  document.getElementById('panel-content').hidden = true;
  document.getElementById('panel').className = 'panel-empty';
}

function renderPanel(d) {
  document.getElementById('panel-placeholder').hidden = true;
  const content = document.getElementById('panel-content');
  content.hidden = false;
  document.getElementById('panel').className = '';

  const kind = d.kind || 'note';
  const badge = document.getElementById('panel-kind-badge');
  badge.textContent = kind;
  badge.className = `badge-${kind}`;

  document.getElementById('panel-label').textContent = d.label || d.id;
  document.getElementById('panel-scope').textContent = `scope: ${d.scope || 'default'}`;
  document.getElementById('panel-id').textContent = d.id;
  document.getElementById('panel-created').textContent = fmtDate(d.created_at);
  document.getElementById('panel-updated').textContent = fmtDate(d.updated_at);

  const contentSection = document.getElementById('panel-content-section');
  if (d.content) {
    contentSection.hidden = false;
    document.getElementById('panel-content-text').textContent = d.content;
  } else {
    contentSection.hidden = true;
  }

  const scoresSection = document.getElementById('panel-scores-section');
  if (d.importance > 0 || d.confidence > 0) {
    scoresSection.hidden = false;
    setBar('importance', d.importance);
    setBar('confidence', d.confidence);
  } else {
    scoresSection.hidden = true;
  }

  const metaSection = document.getElementById('panel-meta-section');
  const metaEntries = Object.entries(d.metadata || {});
  if (metaEntries.length > 0) {
    metaSection.hidden = false;
    const dl = document.getElementById('panel-meta-list');
    dl.innerHTML = '';
    metaEntries.forEach(([k, v]) => {
      const dt = document.createElement('dt'); dt.textContent = k;
      const dd = document.createElement('dd'); dd.textContent = v;
      dl.appendChild(dt);
      dl.appendChild(dd);
    });
  } else {
    metaSection.hidden = true;
  }

  const edgesSection = document.getElementById('panel-edges-section');
  const edges = d.edges || [];
  if (edges.length > 0) {
    edgesSection.hidden = false;
    const ul = document.getElementById('panel-edges-list');
    ul.innerHTML = '';
    edges.forEach(e => {
      const neighborId = e.from_id === d.id ? e.to_id : e.from_id;
      const isOutgoing = e.from_id === d.id;
      const neighborNode = nodeIndex[neighborId];
      const neighborLabel = neighborNode ? truncate(neighborNode.label, 30) : neighborId;

      const li = document.createElement('li');
      li.title = `${e.from_id} → ${e.to_id}`;
      li.addEventListener('click', () => selectNode(neighborId));

      const pill = document.createElement('span');
      pill.className = 'edge-type-pill';
      pill.textContent = e.type;
      pill.style.background = edgeColor(e.type) + '33';
      pill.style.color = edgeColor(e.type);

      const dir = document.createElement('span');
      dir.className = 'edge-dir';
      dir.textContent = isOutgoing ? '→' : '←';

      const lbl = document.createElement('span');
      lbl.className = 'edge-neighbor-label';
      lbl.textContent = neighborLabel;

      li.appendChild(pill);
      li.appendChild(dir);
      li.appendChild(lbl);
      ul.appendChild(li);
    });
  } else {
    edgesSection.hidden = true;
  }
}

function setBar(name, value) {
  const pct = Math.round((value || 0) * 100);
  document.getElementById(`bar-${name}`).style.width = `${pct}%`;
  document.getElementById(`val-${name}`).textContent = pct + '%';
}

function fmtDate(iso) {
  if (!iso) return '—';
  try {
    return new Date(iso).toLocaleString();
  } catch {
    return iso;
  }
}

function truncate(s, n) {
  if (!s) return '';
  return s.length > n ? s.slice(0, n) + '…' : s;
}

function showError(msg) {
  console.error('FireMemory UI:', msg);
}

function cytoscapeStyle() {
  return [
    {
      selector: 'node',
      style: {
        'background-color': ele => nodeColor(ele.data('kind')),
        'label': 'data(label)',
        'color': '#e2e8f0',
        'font-size': 11,
        'text-valign': 'bottom',
        'text-margin-y': 4,
        'text-wrap': 'ellipsis',
        'text-max-width': 120,
        'width': 28,
        'height': 28,
        'border-width': 0,
        'transition-property': 'opacity, border-width',
        'transition-duration': '0.15s',
      },
    },
    {
      selector: 'node.highlighted',
      style: {
        'border-width': 3,
        'border-color': '#ffffff44',
        'width': 34,
        'height': 34,
      },
    },
    {
      selector: 'node.dimmed',
      style: { 'opacity': 0.15 },
    },
    {
      selector: 'edge',
      style: {
        'line-color': ele => edgeColor(ele.data('type')),
        'target-arrow-color': ele => edgeColor(ele.data('type')),
        'target-arrow-shape': 'triangle',
        'curve-style': 'bezier',
        'width': ele => 1 + (ele.data('weight') || 1) * 1.5,
        'opacity': 0.7,
        'transition-property': 'opacity',
        'transition-duration': '0.15s',
      },
    },
    {
      selector: 'edge.dimmed',
      style: { 'opacity': 0.04 },
    },
    {
      selector: 'edge.highlighted',
      style: { 'opacity': 1 },
    },
  ];
}

init();
