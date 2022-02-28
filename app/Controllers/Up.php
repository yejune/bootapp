<?php
namespace App\Controllers;

declare(ticks=1);

class Up extends Command
{
    use \App\Traits\Machine;
    use \App\Traits\Docker\Ls;

    use \App\Traits\Docker\Compose {
        \App\Traits\Docker\Compose::fileGenerate as dockerComposeFileGenerate;
    }

    use \App\Traits\Docker\Run {
        \App\Traits\Docker\Run::Containers as dockerRunContainers;
    }

    /**
     * @var string
     */
    public $loop;
    /**
     * @param $mode
     */
    public $command = 'up';

    /**
     * @param \Peanut\Console\Application $app
     */
    public function configuration(\Peanut\Console\Application $app)
    {
        $app->option('attach', ['require' => false, 'alias' => 'a', 'value' => false]);
        $app->option('pull', ['require' => false, 'alias' => 'p', 'value' => false]);
    }

    /**
     * @param \Peanut\Console\Application $app
     * @param array                       $config
     */
    public function exec(\Peanut\Console\Application $app, array $config)
    {
        $this->process(['sudo', '-v'], ['print' => false]);
        $mode   = $app->getOption('attach') ? 'attach' : 'detach';
        $ispull = $app->getOption('pull') ? true : false;

        if ('attach' == $mode && false === function_exists('pcntl_fork')) {
            $mode = 'detach';
            echo \Peanut\Console\Color::gettext('attach mode is not support, start detach mode', 'red', '').PHP_EOL.PHP_EOL;
        }

        if ('attach' == $mode) {
            $this->loop = \React\EventLoop\Factory::create();
            try {
                $this->run($mode, $ispull);

                $pcntl = new \MKraemer\ReactPCNTL\PCNTL($this->loop);
                $pcntl->on(SIGTERM, function () {
                    // kill
                    echo 'SIGTERM'.PHP_EOL;
                    exit();
                });
                $pcntl->on(SIGHUP, function () {
                    // logout
                    echo 'SIGHUP'.PHP_EOL;
                    exit();
                });
                $pcntl->on(SIGINT, function () {
                    // ctrl+c
                    echo 'Terminated by console'.PHP_EOL;
                    posix_kill(posix_getpid(), SIGUSR1);
                    echo $this->process('docker rm -f $(docker ps -q)', ['print'=>false]);
                    echo PHP_EOL;
                    exit();
                });

                echo 'Started as PID '.getmypid().PHP_EOL;

                $this->loop->run();
            } catch (\Exception $e) {
                throw new \Peanut\Console\Exception($e);
            }
        } else {
            $this->run($mode, $ispull);
            echo PHP_EOL;
            $this->dockerLs();
        }
    }

    public function run($mode, $ispull)
    {
        $this->initMachine();
        $this->dockerComposeFileGenerate();
        $this->setCert();
        $this->dockerRunContainers($mode, $ispull);
        $this->setRoute();
        $this->setHost();
    }
}
