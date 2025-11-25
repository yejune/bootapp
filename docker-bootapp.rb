class DockerBootapp < Formula
  desc "Docker Compose multi-project manager with automatic networking, /etc/hosts, and SSL certificates"
  homepage "https://github.com/yejune/docker-bootapp"
  url "https://github.com/yejune/docker-bootapp/archive/refs/tags/v0.1.1.tar.gz"
  sha256 "0380f0c36a98ddba9bd65d50f7aa4cdf95f09da6f0e72f73576ce369fe79d860"
  license "MIT"
  head "https://github.com/yejune/docker-bootapp.git", branch: "main"

  depends_on "go" => :build

  def install
        system "go", "build", *std_go_args(output: bin/"docker-bootapp"), "."

    # Install to Docker CLI plugins directory
    docker_plugins = "#{ENV["HOME"]}/.docker/cli-plugins"
    mkdir_p docker_plugins
    cp bin/"docker-bootapp", "#{docker_plugins}/docker-bootapp"
    chmod 0755, "#{docker_plugins}/docker-bootapp"

    # Also install as standalone 'bootapp' binary
    bin.install_symlink "docker-bootapp" => "bootapp"

  end

  def test
        assert_match "bootapp", shell_output("#{bin}/docker-bootapp help")
    assert_match "bootapp", shell_output("#{bin}/bootapp help")

  end

  def caveats
    <<~EOS
            docker-bootapp has been installed!
      
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
      
      Documentation: https://github.com/yejune/docker-bootapp
      
    EOS
  end
end
