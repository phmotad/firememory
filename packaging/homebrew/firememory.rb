# Homebrew formula for FireMemory.
#
# This file belongs in the tap repository: phmotad/homebrew-firememory
# Path: Formula/firememory.rb
#
# To publish: copy this file into that repo and fill in the real sha256 and
# version values after the first `v*` GitHub Release is created.
#
# Users install via:
#   brew tap phmotad/firememory
#   brew install firememory

class Firememory < Formula
  desc "Local-first semantic memory engine for AI agents"
  homepage "https://github.com/phmotad/firememory"
  version "0.1.1"
  license "Apache-2.0"

  on_macos do
    on_arm do
      url "https://github.com/phmotad/firememory/releases/download/v#{version}/firememory_#{version}_darwin_arm64.tar.gz"
      sha256 "5b531622b798dedc490b95341769ff7f480d1278e54f6b4720ea43883ee43e9c"
    end
    on_intel do
      url "https://github.com/phmotad/firememory/releases/download/v#{version}/firememory_#{version}_darwin_amd64.tar.gz"
      sha256 "4065d4fa95e92f87cd2042c6bd375d8e7b45aab95438cdd57ad664c7356ef9fb"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/phmotad/firememory/releases/download/v#{version}/firememory_#{version}_linux_arm64.tar.gz"
      sha256 "87a9b3d28830dd93dce9033980fe46fd8b43ee60c70351552544ccd138513d57"
    end
    on_intel do
      url "https://github.com/phmotad/firememory/releases/download/v#{version}/firememory_#{version}_linux_amd64.tar.gz"
      sha256 "85191759a6629c1af19ad3a61a9fe17a6bc8539755785868178ab80dbc4ba439"
    end
  end

  def install
    bin.install "fmem"

    # The ONNX Runtime shared library must live next to fquery (or be on the
    # dynamic linker search path). We install it to lib/ and wrap fquery with a
    # small shell script that sets FIREMEMORY_ORT_LIB_PATH.
    ort_lib = Dir["libonnxruntime*.dylib", "libonnxruntime*.so"].first
    if ort_lib
      lib.install ort_lib
      ort_lib_path = "#{lib}/#{ort_lib}"
      (bin/"fquery-bin").install "fquery"
      (bin/"fquery").write <<~SH
        #!/bin/sh
        export FIREMEMORY_ORT_LIB_PATH="#{ort_lib_path}"
        exec "#{bin}/fquery-bin" "$@"
      SH
      chmod 0755, bin/"fquery"
    else
      bin.install "fquery"
    end
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/fmem version 2>&1", 1)
    system "#{bin}/fquery", "doctor"
  end
end
