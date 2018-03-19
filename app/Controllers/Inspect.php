<?php
namespace App\Controllers;

class Inspect extends Command
{
    use \App\Traits\Machine;
    /**
     * @var string
     */
    public $command = 'inspect';

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
        $this->dockerInspect($name);
    }

    /**
     * @param $name
     */
    public function dockerInspect($name)
    {
        $containerName = $this->getContainerName($name);

        $command = [
            'docker',
            'inspect',
            $containerName,
        ];

        echo 'command | ';
        echo \Peanut\Console\Color::gettext(implode(' ', $command), 'white').PHP_EOL.PHP_EOL;
        echo $this->process($command, ['print' => false]);
    }
}
