<?php
namespace App\Controllers;

class Unpause extends Command
{
    use \App\Traits\Machine;
    /**
     * @var string
     */
    public $command = 'unpause';

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
        $this->dockerUnpause($name);
    }

    /**
     * @param $name
     */
    public function dockerUnpause($name)
    {
        $containerName = $this->getContainerName($name);

        $command = [
            'docker',
            'unpause',
            $containerName,
        ];

        echo 'command | ';
        echo \Peanut\Console\Color::gettext(implode(' ', $command), 'white').PHP_EOL.PHP_EOL;
        echo $this->process($command, ['print' => false]);
    }
}
