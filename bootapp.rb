class Bootapp < Formula
  desc "Docker Compose multi-project manager with automatic networking, /etc/hosts, and SSL certificates"
  homepage "https://github.com/yejune/bootapp"
  url "https://github.com/yejune/bootapp/archive/refs/tags/v1.0.15.tar.gz"
  sha256 "6e611e6b871ed4dd4f5d7468334d08a507336e68049cf6adfc9c3e67f9ebe4dd"
  license "MIT"
  head "https://github.com/yejune/bootapp.git", branch: "main"

  depends_on "go" => :build

  def install
    ldflags = "-s -w -X github.com/yejune/bootapp/cmd.Version=#{version}"
    system "go", "build", *std_go_args(ldflags:, output: bin/"bootapp"), "."

    # Install to Docker CLI plugins directory (must be docker-<name> format)
    docker_plugins = "#{ENV["HOME"]}/.docker/cli-plugins"
    mkdir_p docker_plugins
    cp bin/"bootapp", "#{docker_plugins}/docker-bootapp"
    chmod 0755, "#{docker_plugins}/docker-bootapp"

  end

  def test
        assert_match "bootapp", shell_output("#{bin}/bootapp help")
    assert_match "bootapp", shell_output("#{bin}/bootapp help")

  end

  def caveats
    <<~EOS
            bootapp has been installed!
      
      Features:
        ✓ Automatic unique subnet allocation per project
        ✓ Auto /etc/hosts management (domain → container IP)
        ✓ SSL certificate generation and system trust
        ✓ Support for multiple Docker Compose projects simultaneously
        ✓ Works with macOS (OrbStack/Docker Desktop/Colima) and Linux
      
      You can use it in two ways:
        docker bootapp [command]  # As Docker CLI plugin
        bootapp [command]         # As standalone binary
      
      Quick Start:
        cd your-docker-compose-project
        docker bootapp up         # Start with auto-networking
        docker bootapp down       # Stop and cleanup
        docker bootapp ls         # List managed projects
      
      Documentation: https://github.com/yejune/bootapp
      
    EOS
  end
end
