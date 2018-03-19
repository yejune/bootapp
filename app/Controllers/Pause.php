<?php
namespace App\Controllers;

class Pause extends Command
{
    use \App\Traits\Machine;
    /**
     * @var string
     */
    public $command = 'pause';

    /**
     * @param \Peanut\Console\Application $app
     */
    public function configuration(\Peanut\Console\Application $app)
    {
        $app->argument('container name');
    }

    /**
     * @param \Peanut\Console\Application $app
     * @param array                       $config
     */
    public function exec(\Peanut\Console\Application $app, array $config)
    {
        $name = $app->getArgument('container name');

        $this->initMachine();
        $this->dockerPause($name);
    }

    /**
     * @param $name
     */
    public function dockerPause($name)
    {
        $containerName = $this->getContainerName($name);

        $command = [
            'docker',
            'pause',
            $containerName,
        ];

        echo 'command | ';
        echo \Peanut\Console\Color::gettext(implode(' ', $command), 'white').PHP_EOL.PHP_EOL;
        echo $this->process($command, ['print' => false]);
    }
}
