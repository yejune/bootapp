<?php
namespace App\Controllers;

class Rm extends Command
{
    use \App\Traits\Machine;

    /**
     * @var string
     */
    public $command = 'rm';

    /**
     * @param \Peanut\Console\Application $app
     */
    public function configuration(\Peanut\Console\Application $app)
    {
        $app->argument('container name');
        $app->option('force', ['require' => false, 'alias' => 'f', 'value' => false]);
    }

    /**
     * @param \Peanut\Console\Application $app
     * @param array                       $config
     */
    public function exec(\Peanut\Console\Application $app, array $config)
    {
        $name  = $app->getArgument('container name');
        $force = $app->getOption('force');

        $this->initMachine();
        $this->dockerRm($name, $force);
    }

    /**
     * @param $name
     * @param $force
     */
    public function dockerRm($name, $force = false)
    {
        $containerName = $this->getContainerName($name);

        $command = [
            'docker',
            'rm',
        ];

        if ($force) {
            $command[] = '-f';
        }

        $command[] = $containerName;

        echo 'command | ';
        echo \Peanut\Console\Color::gettext(implode(' ', $command), 'white').PHP_EOL.PHP_EOL;
        $this->process($command, ['print' => true]);
    }
}
