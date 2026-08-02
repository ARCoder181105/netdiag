# NOT PUBLISHED YET.
#
# This formula is a template. Before it can be used:
#   1. Create a tap repository named `homebrew-netdiag` under the same owner.
#   2. Bump `version` and the four release URLs to the target tag.
#   3. Replace every PLACEHOLDER_SHA256_* with the real value from the
#      release's checksums.txt.
#
# The README deliberately does not advertise `brew install` until that is done.
class Netdiag < Formula
  desc "Powerful, unified network diagnostic CLI tool"
  homepage "https://github.com/ARCoder181105/netdiag"
  version "0.1.0"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/ARCoder181105/netdiag/releases/download/v0.1.0/netdiag-darwin-arm64"
      sha256 "PLACEHOLDER_SHA256_ARM64"
    else
      url "https://github.com/ARCoder181105/netdiag/releases/download/v0.1.0/netdiag-darwin-amd64"
      sha256 "PLACEHOLDER_SHA256_AMD64"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/ARCoder181105/netdiag/releases/download/v0.1.0/netdiag-linux-arm64"
      sha256 "PLACEHOLDER_SHA256_LINUX_ARM64"
    else
      url "https://github.com/ARCoder181105/netdiag/releases/download/v0.1.0/netdiag-linux-amd64"
      sha256 "PLACEHOLDER_SHA256_LINUX_AMD64"
    end
  end

  def install
    # The downloaded file is the binary itself, just needs to be renamed
    bin.install Dir["netdiag-*"].first => "netdiag"
  end

  def caveats
    <<~EOS
      For ICMP operations (ping, trace, discover) on Linux, run:
        sudo setcap cap_net_raw+ep #{bin}/netdiag

      On macOS, ICMP operations require sudo:
        sudo netdiag ping google.com
        sudo netdiag trace github.com
        sudo netdiag discover

      Other commands (scan, http, dig, whois, speedtest) work without sudo.
    EOS
  end

  test do
    assert_match "netdiag", shell_output("#{bin}/netdiag --help")
  end
end
