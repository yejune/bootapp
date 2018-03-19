<?php
namespace App\Controllers;

class Task extends Command
{
    use \App\Traits\Machine;

    /**
     * @var string
     */
    public $command = 'task';

    /**
     * @param \Peanut\Console\Application $app
     */
    public function configuration(\Peanut\Console\Application $app)
    {
        $app->argument('task name');
        $app->all();
    }

    /**
     * @param \Peanut\Console\Application $app
     * @param array                       $config
     */
    public function exec(\Peanut\Console\Application $app, array $config)
    {
        $this->initMachine();

        $taskName = $app->getArgument('task name');
        $action   = $app->getArgument('*');

        if (true === isset($config['tasks'])) {
            if (true === in_array($taskName, array_keys($config['tasks']))) {
                $this->dockerTask($config['tasks'][$taskName], $action);
            } else {
                throw new \Peanut\Console\Exception($taskName.' not defined');
            }
        }
    }

    /**
     * @param $task
     * @param $action
     */
    public function dockerTask($task, $action)
    {
        $containerName = $this->getContainerName($task['container']);

        $command = [
            'docker',
            'exec',
        ];

        if (true === isset($task['user'])) {
            $command[] = '--user='.$task['user'];
        }

        $command[] = '-it';
        $command[] = $containerName;
        $command[] = 'sh -c';
        $command[] = '"';

        if (true === isset($task['working_dir'])) {
            $command[] = 'cd '.$task['working_dir'];
            $command[] = '&&';
        }

        $command[] = $task['cmd'];

        if ($action) {
            $command[] = $action;
        }

        $command[] = '"';

        echo 'command | ';
        echo \Peanut\Console\Color::gettext(implode(' ', $command), 'white').PHP_EOL.PHP_EOL;
        $this->process($command, ['print' => true, 'tty' => true]);
    }
}
