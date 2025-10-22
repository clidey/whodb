cask "whodb" do
  version "VERSION_PLACEHOLDER"
  sha256 "SHA256_PLACEHOLDER"

  url "https://github.com/clidey/whodb/releases/download/v#{version}/whodb.dmg"
  name "WhoDB"
  desc "Modern database management and visualization tool with AI integration"
  homepage "https://whodb.com"

  auto_updates true

  app "WhoDB.app"

  livecheck do
    url "https://github.com/clidey/whodb/releases.atom"
    regex(/href=.*?\/v?(\d+(?:\.\d+)*)\//i)
    strategy :github_latest
  end

  uninstall quit: "com.clidey.whodb.ce",
            signal: ["TERM", "com.clidey.whodb.ce"]

  zap trash: [
    "~/Library/Application Support/com.clidey.whodb.ce",
    "~/Library/Caches/com.clidey.whodb.ce",
    "~/Library/Preferences/com.clidey.whodb.ce.plist",
    "~/Library/Saved Application State/com.clidey.whodb.ce.savedState",
  ]

  caveats do
    "WhoDB is a database management tool. Visit https://whodb.com for documentation."
  end
end
