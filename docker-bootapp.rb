class DockerBootapp < Formula
  desc "Docker CLI Plugin for multi-project Docker networking made easy"
  homepage "https://github.com/yejune/docker-bootapp"
  url "https://github.com/yejune/docker-bootapp/archive/refs/tags/v1.0.0.tar.gz"
  sha256 "503cbab3bb70a539c2d30f8636af14995168a0ae6fff8649834fdb6cf6258ab4"
  license "MIT"
  head "https://github.com/yejune/docker-bootapp.git", branch: "main"

  depends_on "go" => :build

  def install
    # Build the binary
    system "go", "build", *std_go_args(output: bin/"docker-bootapp"), "."

    # Install to Docker CLI plugins directory
    docker_plugins = "#{ENV["HOME"]}/.docker/cli-plugins"
    mkdir_p docker_plugins
    cp bin/"docker-bootapp", "#{docker_plugins}/docker-bootapp"
    chmod 0755, "#{docker_plugins}/docker-bootapp"

    # Also install as standalone 'bootapp' binary
    bin.install_symlink "docker-bootapp" => "bootapp"
  end

  def caveats
    <<~EOS
      docker-bootapp has been installed!

      You can use it in two ways:
        docker bootapp [command]  # As Docker CLI plugin
        bootapp [command]         # As standalone binary

      Usage:
        docker bootapp up      # Start containers with auto-networking
        docker bootapp down    # Stop containers
        docker bootapp ls      # List managed projects

      On macOS with Docker Desktop/Colima, docker-mac-net-connect is recommended:
        brew install chipmk/tap/docker-mac-net-connect
        sudo brew services start docker-mac-net-connect

      On macOS with OrbStack or Linux, no additional tools needed!
    EOS
  end

  test do
    # Test that both binaries work
    assert_match "bootapp", shell_output("#{bin}/docker-bootapp help")
    assert_match "bootapp", shell_output("#{bin}/bootapp help")

    # Verify it's installed as a Docker plugin
    docker_plugin = "#{ENV["HOME"]}/.docker/cli-plugins/docker-bootapp"
    assert_predicate Pathname.new(docker_plugin), :exist?
  end
end
