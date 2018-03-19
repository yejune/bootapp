<?php
namespace App\Controllers;

class Log extends Command
{
    use \App\Traits\Machine;
    /**
     * @var string
     */
    public $command = 'log';

    /**
     * @param \Peanut\Console\Application $app
     */
    public function configuration(\Peanut\Console\Application $app)
    {
        $app->option('follow', ['require' => false, 'alias' => 'f', 'value' => false]);
        $app->argument('container name');
    }

    /**
     * @param \Peanut\Console\Application $app
     * @param array                       $config
     */
    public function exec(\Peanut\Console\Application $app, array $config)
    {
        $isFollow = $app->getOption('follow');
        $name     = $app->getArgument('container name');
        $this->initMachine();
        $this->dockerLog($name, $isFollow);
    }

    /**
     * @param $name
     * @param mixed $isFollow
     */
    public function dockerLog($name, $isFollow = false)
    {
        $containerName = $this->getContainerName($name);

        $command = [
            'docker',
            'logs',
        ];

        if ($isFollow) {
            $command[] = '-f';
        }

        $command[] = $containerName;

        echo 'command | ';
        echo \Peanut\Console\Color::gettext(implode(' ', $command), 'white').PHP_EOL.PHP_EOL;

        //echo shell_exec(implode(' ', $command));
        $this->process($command, ['print' => true]);
    }
}
