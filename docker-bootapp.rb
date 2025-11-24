class DockerBootapp < Formula
  desc "Docker CLI Plugin for multi-project Docker networking made easy"
  homepage "https://github.com/yejune/docker-bootapp"
  url "https://github.com/yejune/docker-bootapp/archive/refs/tags/v1.0.0.tar.gz"
  sha256 "" # Will be updated when you create a release
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
  end

  def caveats
    <<~EOS
      docker-bootapp has been installed as a Docker CLI plugin!

      Usage:
        docker bootapp up      # Start containers with auto-networking
        docker bootapp down    # Stop containers
        docker bootapp ls      # List managed projects

      On macOS, docker-mac-net-connect is required for direct container IP access:
        brew install chipmk/tap/docker-mac-net-connect
        sudo brew services start docker-mac-net-connect
    EOS
  end

  test do
    # Test that the binary works
    assert_match "docker-bootapp", shell_output("#{bin}/docker-bootapp --help")

    # Verify it's installed as a Docker plugin
    docker_plugin = "#{ENV["HOME"]}/.docker/cli-plugins/docker-bootapp"
    assert_predicate Pathname.new(docker_plugin), :exist?
  end
end
