<?php
namespace App\Controllers;

use Symfony\Component\Process\Process;
use Herrera\Phar\Update\Manager;
use Herrera\Phar\Update\Manifest;
use Herrera\Version\Parser;

class Command extends \Peanut\Console\Command
{
    /**
     * @var string
     */
    public $configFileName = 'Bootfile.yml';

    /**
     * @var string
     */
    public $color = '';

    /**
     * @var mixed
     */

    public $verbose = false;

    public $cwd;

    /**
     * @param $command
     * @param $option
     */
    public function process($command, array $option = null)
    {
        if (true === is_array($command)) {
            $command = implode(' ', $command);
        }

        if (true === isset($option['timeout'])) {
            $timeout = $option['timeout'];
        } else {
            $timeout = null;
        }

        if (true === isset($option['tty'])) {
            $tty = $option['tty'];
        } else {
            $tty = false;
        }

        if ($this->verbose) {
            $print = true;
            $this->message(\Peanut\Console\Color::gettext('IN >> ', 'white').\Peanut\Console\Color::gettext($command, 'red'));
        } else {
            if (true === isset($option['print'])) {
                $print = $option['print'];
            } else {
                $print = true;
                $this->message($command);
            }
        }

        $process = new Process($command);
        $process->setTty($tty);
        $process->setTimeout($timeout);
        $printCount = 0;
        $process->run(function ($type, $buf) use ($print, $command, &$printCount) {
            if (true === $print) {
                $buffers = explode(PHP_EOL, trim($buf, PHP_EOL));

                foreach ($buffers as $buffer) {
                    if (Process::ERR === $type) {
                        echo 'ERR > '.$buffer.PHP_EOL;
                    } else {
                        if ('reach it successfully.' == $buffer) {
                            print_r('Check your docker-machine.');
                        }

                        echo \Peanut\Console\Color::gettext('OUT > ', 'black').\Peanut\Console\Color::gettext($buffer, 'dark_gray').PHP_EOL;
                    }
                }
            } elseif ('.' == $print) {
                if ($printCount > 0) {
                    echo $print;
                }
            }
            $printCount ++;
        });

        if ($process->getExitCode() && $process->getErrorOutput()) {
            // $traces = debug_backtrace();
            // print_r($traces[0]);
            $msg = trim($process->getErrorOutput()).'('.$process->getExitCode().')';
            throw new \Peanut\Console\Exception($msg);
        }

        return new \Peanut\Console\Result($process->getErrorOutput().$process->getOutput());
    }

    /**
     * @param $name
     * @param $command
     */
    public function childProcess($name, $command)
    {
        $process = new \React\ChildProcess\Process($command);

        $process->on('exit', function ($exitCode, $termSignal) {
            // ...
            $this->message($exitCode, 'exit');
            $this->message($termSignal, 'exit');
        });

        if (!$this->color) {
            $tmp = \Peanut\Console\Color::$foregroundColors;
            unset($tmp['black'], $tmp['default'], $tmp['white'], $tmp['light_gray']);
            $this->color = array_keys($tmp);
        }

        $color = array_shift($this->color);

        $this->loop->addTimer(0.001, function ($timer) use ($process, $name, $color) {
            $process->start($timer->getLoop());

            $callback = function ($output) use ($name, $color) {
                $lines = explode(PHP_EOL, $output);
                $i     = 0;

                foreach ($lines as $line) {
                    if ($line) {
                        $tmp = '';

                        if ($name) {
                            $tmp .= \Peanut\Console\Color::gettext(str_pad($name, 16, ' ', STR_PAD_RIGHT).' | ', $color);
                        }

                        $tmp .= \Peanut\Console\Color::gettext($line, 'light_gray');

                        $this->message($tmp);
                    }

                    $i++;
                }
            };
            $process->stdout->on('data', $callback);
            $process->stderr->on('data', $callback);
        });
    }

    /**
     * @param $message
     * @param $name
     */
    public function message($message = '', $name = '')
    {
        $this->log($message, $name);
    }

    /**
     * @param $m
     * @param mixed $message
     * @param mixed $name
     */
    public function log($message = '', $name = '')
    {
        if (true === is_array($message)) {
            foreach ($message as $k => $v) {
                $this->log($v, $k);
            }
        } else {
            if ($name && false === is_numeric($name)) {
                echo $name;
                echo ' = ';
            }

            echo($message).PHP_EOL;
        }
    }

    /**
     * @param  $data
     * @return mixed
     */
    public function table($data)
    {
        // Find longest string in each column
        $columns = [];

        foreach ($data as $row_key => $row) {
            $i = -1;

            foreach ($row as $cell) {
                $i++;
                $length = strlen($cell);

                if (empty($columns[$i]) || $columns[$i] < $length) {
                    $columns[$i] = $length;
                }
            }
        }

        $ret = [];

        foreach ($data as $row_key => $row) {
            $i     = -1;
            $table = '';

            foreach ($row as $cell) {
                $i++;
                $table .= str_pad($cell ?: ' ', $columns[$i], ' ').'   ';
            }

            $ret[] = $table;
        }

        return $ret;
    }

    /**
     * @return string
     */
    public function getStageName()
    {
        if (true === isset($this->config['stage_name'])) {
            return $this->config['stage_name'];
        }

        return 'local';
    }

    /**
     * @return string
     */
    public function getMachineName()
    {
        if (true === isset($this->config['machine_name'])) {
            return $this->config['machine_name'];
        }

        return 'bootapp-docker-machine';
    }

    /**
     * @return string
     */
    public function getProjectName()
    {
        if (true === isset($this->config['project_name'])) {
            return $this->config['project_name'];
        }

        return '';
    }

    public function getMachineEnvironment()
    {
        if (true === isset($this->config['environment'])) {
            return $this->config['environment'];
        }

        return [];
    }

    public function getMachineEnvironmentMemorySize()
    {
        if (true === isset($this->config['environment']['memory_size'])) {
            return $this->config['environment']['memory_size'];
        }

        return 0;
    }

    public function getMachineEnvironmentInitScripts()
    {
        if (true === isset($this->config['environment']['init_scripts'])) {
            return $this->config['environment']['init_scripts'];
        }

        return [];
    }

    /**
     * @return string
     * @param mixed $name
     */
    public function getContainerName($name)
    {
        return $this->getProjectName().'-'.$name;
    }

    /**
     * @param  $message
     * @return string
     */
    public function ask($message)
    {
        echo $message;
        $handle = fopen('php://stdin', 'r');
        $line   = fgets($handle);
        fclose($handle);

        return trim($line);
    }

    /**
     * @param  \Peanut\Console\Application $app
     * @return mixed
     */
    public function execute(\Peanut\Console\Application $app)
    {
        $version = $app->getApplicationVersion();
        $update  = $app->getOption('no-update') ? 'no' : '';

        // 인터넷에 연결되었는지 확인
        $ip = shell_exec('dig +short myip.opendns.com @resolver1.opendns.com 2> /dev/null');
        if (!$ip) {
            $update = 'no';
        }

        if (version_compare($version, '0.0.0') > 0 && $update != 'no') {
            $manager = new Manager($manifest = Manifest::loadFile(
                'https://raw.githubusercontent.com/yejune/bootapp/master/manifest.json'
            ));

            $update = $manifest->findRecent(
                Parser::toVersion($version),
                true,
                true
            );
            if ($this->verbose) {
                $print = true;
                $this->message(\Peanut\Console\Color::gettext('IN >> ', 'white').\Peanut\Console\Color::gettext($update, 'red'));
                echo $update;
            }
            if (null !== $update) {
                echo \Peanut\Console\Color::gettext(((string)$update).' New version is available.', 'white', 'red').PHP_EOL;
                echo \Peanut\Console\Color::gettext('Please execute `bootapp self-update` Or use --no-update(-n) option', 'white', 'red').PHP_EOL;
                exit;
            }
        }

        $this->config  = $this->getConfig();
        $this->verbose = $app->getOption('verbose');

        return $this->exec($app, $this->config);
    }

    /**
     * @return array
     */
    public function setConfig()
    {
        $this->config = \App\Helpers\Yaml::parseFile($this->configFileName);
        $this->cwd    = \App\Helpers\Yaml::$cwd;
    }

    /**
     * @return array
     */
    public function getConfig()
    {
        if (false === isset($this->config)) {
            $this->setConfig();
        }

        return $this->config;
    }
    public function isLinux()
    {
        if ('LINUX' == strtoupper(PHP_OS)) {
            return true;
        }
        return false;
    }
    /**
     * @return string
     */
    public function getMachineIp()
    {
        if ($this->isLinux()) {
            return '';
        }
        return $this->process([
            'docker-machine',
            'ip',
            $this->getMachineName(),
        ], ['print' => false]); //parse_url(getenv('DOCKER_HOST'), PHP_URL_HOST);
    }

    public function getcwd()
    {
        return $this->cwd;
    }
}
