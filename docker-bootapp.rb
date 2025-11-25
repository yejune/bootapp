class DockerBootapp < Formula
  desc "Docker Compose multi-project manager with automatic networking, /etc/hosts, and SSL certificates"
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
      🎉 docker-bootapp has been installed!

      Features:
        ✓ Automatic unique subnet allocation per project
        ✓ Auto /etc/hosts management (domain → container IP)
        ✓ SSL certificate generation and system trust
        ✓ Support for multiple Docker Compose projects simultaneously
        ✓ Works with macOS (OrbStack/Docker Desktop/Colima) and Linux

      You can use it in two ways:
        docker bootapp [command]  # As Docker CLI plugin
        bootapp [command]         # As standalone binary

      Available Commands:
        up          Create and start containers with network setup
                    Flags: -d (detach), -F (force-recreate), --no-build, --pull
        down        Stop and remove containers
        ls          List registered projects
        cert        Manage SSL certificates
                    Subcommands: detect, generate, install, list, uninstall
        install     Install docker-bootapp as a Docker CLI plugin
        completion  Generate shell completion (bash, zsh, fish, powershell)

      Global Flags:
        -f, --file   Compose file path (default: auto-detect)

      Environment Variables (in docker-compose.yml):
        DOMAIN=myapp.test                    # Single domain
        SSL_DOMAINS=app.test,api.test       # Multiple domains with SSL
        DOMAINS=web.test,admin.test         # Multiple domains without SSL

      Platform-specific requirements:
        • macOS + Docker Desktop/Colima: docker-mac-net-connect recommended
          brew install chipmk/tap/docker-mac-net-connect
          sudo brew services start docker-mac-net-connect

        • macOS + OrbStack: No additional tools needed! ✓
        • Linux: No additional tools needed! ✓

      Documentation: https://github.com/yejune/docker-bootapp
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
