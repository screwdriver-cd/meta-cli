# typed: false
# frozen_string_literal: true

# This file was generated by GoReleaser. DO NOT EDIT.
class MetaCli < Formula
  desc "CLI for reading/writing Screwdriver project metadata"
  homepage "https://github.com/screwdriver-cd/meta-cli"
  version "0.0.81"

  on_macos do
    if Hardware::CPU.intel?
      url "https://github.com/screwdriver-cd/meta-cli/releases/download/v0.0.81/meta-cli_darwin_amd64"
      sha256 "743d21fbc2ceee2ef12b5d667504402592019b3ece6604cf20b229362c4ac06d"

      def install
        bin.install File.basename(@stable.url) => "meta"
        ohai 'Notice', <<~EOL
          In order to use, you may wish to add the following to your ~/.bash_profile and execute now

            export SD_META_DIR="$HOME/meta"
            mkdir -p "$SD_META_DIR"

        EOL
      end
    end
    if Hardware::CPU.arm?
      url "https://github.com/screwdriver-cd/meta-cli/releases/download/v0.0.81/meta-cli_darwin_arm64"
      sha256 "1db13a09dd81e4b3b3b6bac58fb6db4ffa448fa8c9a730c18848d917546b2298"

      def install
        bin.install File.basename(@stable.url) => "meta"
        ohai 'Notice', <<~EOL
          In order to use, you may wish to add the following to your ~/.bash_profile and execute now

            export SD_META_DIR="$HOME/meta"
            mkdir -p "$SD_META_DIR"

        EOL
      end
    end
  end

  on_linux do
    if Hardware::CPU.arm? && Hardware::CPU.is_64_bit?
      url "https://github.com/screwdriver-cd/meta-cli/releases/download/v0.0.81/meta-cli_linux_arm64"
      sha256 "0b9b91951e3b1230b87c4050bf76d4fd7c0d01c6eccd561ee0c5f00b0e4e81c6"

      def install
        bin.install File.basename(@stable.url) => "meta"
        ohai 'Notice', <<~EOL
          In order to use, you may wish to add the following to your ~/.bash_profile and execute now

            export SD_META_DIR="$HOME/meta"
            mkdir -p "$SD_META_DIR"

        EOL
      end
    end
    if Hardware::CPU.intel?
      url "https://github.com/screwdriver-cd/meta-cli/releases/download/v0.0.81/meta-cli_linux_amd64"
      sha256 "5d6badb67e28d1dee8fadbae491b9e63c302b6647aaf13e87b55c8fcd1e64918"

      def install
        bin.install File.basename(@stable.url) => "meta"
        ohai 'Notice', <<~EOL
          In order to use, you may wish to add the following to your ~/.bash_profile and execute now

            export SD_META_DIR="$HOME/meta"
            mkdir -p "$SD_META_DIR"

        EOL
      end
    end
  end

  test do
    system "#{bin}/meta-cli", "--version"
  end
end
