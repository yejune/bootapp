<?php
namespace App\Controllers;

class Ssh extends Command
{
    use \App\Traits\Machine;
    /**
     * @var string
     */
    public $command = 'ssh';

    /**
     * @param \Peanut\Console\Application $app
     */
    public function configuration(\Peanut\Console\Application $app)
    {
        $app->argument('container name');
        $app->all();
    }

    /**
     * @param \Peanut\Console\Application $app
     * @param array                       $config
     */
    public function exec(\Peanut\Console\Application $app, array $config)
    {
        $name    = $app->getArgument('container name');
        $command = $app->getArgument('*');

        $this->initMachine();
        $this->dockerSsh($name, $command);
    }

    /**
     * @param $name
     * @param $cmd
     */
    public function dockerSsh($name, $cmd = '')
    {
        $containerName = $this->getContainerName($name);

        if (!$cmd) {
            $cmd = 'sh';
        }

        $command = [
            'docker',
            'exec',
            '-it',
            $containerName,
            $cmd,
        ];

        echo 'command | ';
        echo \Peanut\Console\Color::gettext(implode(' ', $command), 'white').PHP_EOL.PHP_EOL;
        $this->process($command, ['print' => false, 'tty' => true]);
    }
}
