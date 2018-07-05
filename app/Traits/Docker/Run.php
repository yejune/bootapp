<?php
namespace App\Traits\Docker;

trait Run
{
    /**
     * @var string
     */
    public $networkName = ['bridge'];

    /**
     * @return mixed
     */
    public function getNetworkList()
    {
        $command = [
            'docker',
            'network',
            'inspect',
            '--format="name={{.Name}}&subnet={{range .IPAM.Config}}{{.Subnet}}{{end}}"',
            '$(docker network ls -q)',
            '2>&1',
        ];

        $raw = $this->process($command, ['print' => false])->toArray();

        $networks = [];

        foreach ($raw as $network) {
            parse_str($network, $out);

            if (true === isset($out['name']) && $out['name']
                && true === isset($out['subnet']) && $out['subnet']) {
                $networks[$out['name']] = $out['subnet'];
            }
        }

        return $networks;
    }

    /**
     * @param $path
     */
    public function volumeRealPath($path)
    {
        $cwd = $this->getcwd();

        $real = preg_replace([
            '#^.(/|:)#',
            '#(\s)#',
        ], [
            $cwd.'$1',
            '\\ ',
        ], $path);

        if (0 === preg_match('#:r(o|w)$#', $path, $match)) {
            $real .= ':rw';
        }

        return $real;
    }

    /**
     * @param mixed $mode
     * @param mixed $isPull
     */
    public function Containers($mode, $isPull)
    {
        $machineName = $this->getMachineName();
        $projectName = $this->getProjectName();

        if (!$projectName) {
            echo PHP_EOL;

            while (true) {
                $inputProjectName = $this->ask('Please project a name : ');

                $command = [
                    'docker',
                    'ps',
                    '-aq',
                    '--filter="label=com.docker.bootapp.project='.$inputProjectName.'"',
                ];
                $existsCount = count($this->process($command, ['print' => false])->toArray());

                if (0 == $existsCount) {
                    break;
                }
                echo 'Name invalid. ';
            }

            $this->config = ['project_name' => $inputProjectName] + $this->config;
            \App\Helpers\Yaml::dumpFile($this->configFileName, $this->config);
            $projectName = $this->getProjectName();
        }

        $stageName   = $this->getStageName();
        $machineName = $this->getMachineName();
        $compose     = \App\Helpers\Yaml::parseFile($this->getcwd().'/docker-compose.'.$stageName.'.yml');

        echo \Peanut\Console\Color::gettext('machine | ', 'white').$machineName.PHP_EOL;
        echo \Peanut\Console\Color::gettext('project | ', 'white').$projectName.PHP_EOL;
        echo \Peanut\Console\Color::gettext('stage   | ', 'white').$stageName.PHP_EOL;

        $checkBuild = false;
        $checkPull  = false;
        // name setting
        {
            foreach ($compose['services'] as $serviceName => &$service) {
                if (true === isset($service['name'])) {
                    $name = $service['name'];
                } else {
                    $name = $serviceName;
                }

                $service['org_name'] = $name;
                $service['name']     = $this->getContainerName($name);

                if (true === isset($service['build'])) {
                    $checkBuild = true;
                } elseif (true === isset($service['image'])) {
                    $checkPull = true;
                }
            }

            // break the reference with the last element
            unset($service);
        }

        // pull
        {
            if ($checkPull && $isPull) {
                echo \Peanut\Console\Color::gettext('pull    | ', 'white');

                foreach ($compose['services'] as $serviceName => $service) {
                    if (true === isset($service['image'])) {
                        $command = [
                            'docker',
                            'pull',
                            $service['image'],
                        ];
                        //$this->message('build '.$service['name']);
                        echo $service['org_name'];
                        $this->process($command, ['print' => '.']);
                        echo ' ';
                    }
                }

                echo PHP_EOL;
            }
        }

        // build
        {
            if ($checkBuild) {
                echo \Peanut\Console\Color::gettext('build   | ', 'white');

                foreach ($compose['services'] as $serviceName => $service) {
                    if (true === isset($service['build'])) {
                        $buildOpts   = [];
                        $buildOpts[] = 'docker';
                        $buildOpts[] = 'build';
                        $buildOpts[] = '--no-cache';
                        $buildOpts[] = '--pull';
                        $buildOpts[] = '--tag='.$service['name'];

                        if (true === is_array($service['build'])) {
                            if (true === isset($service['build']['args'])) {
                                foreach ($service['build']['args'] as $argKey => $argValue) {
                                    $buildOpts[] = '--build-arg '.$argKey.'='.escapeshellarg($argValue);
                                }
                            }

                            if (true === isset($service['build']['context'])) {
                                $buildOpts[] = $service['build']['context'];
                            } else {
                                throw new \Console\Exception('build context not found');
                            }

                            if (true === isset($service['build']['dockerfile'])) {
                                $buildOpts[] = '-f '.$service['build']['dockerfile'];
                            }
                        } else {
                            $buildOpts[] = $service['build'];
                        }

                        //$this->message('build '.$service['name']);
                        echo $service['org_name'];
                        $this->process($buildOpts, ['print' => '.']);
                        echo ' ';
                    }
                }

                echo PHP_EOL;
            }
        }

        // remove
        {
            echo \Peanut\Console\Color::gettext('remove  | ', 'white');

            foreach ($compose['services'] as $serviceName => $service) {
                echo $service['org_name'].' ';

                $rmCommand = [
                    'docker',
                    'rm',
                    '-f',
                    $service['name'],
                    '2>&1',
                ];
                $this->process($rmCommand, ['print' => false]);
            }

            echo PHP_EOL;
        }

        // network
        {
            if (true === isset($compose['networks'])) {
            } else {
                // default network setting
                $compose['networks'] = [
                    'default' => [
                        'ipam' => [
                            'config' => [
                                ['subnet'],
                            ],
                        ],
                    ],
                ];
            }

            if (true === isset($compose['networks'])) {
                $defaultName = $projectName;

                foreach ($compose['networks'] as $networkName => $network) {
                    $dockerNetworks = $this->getNetworkList();

                    if ('default' == $networkName) {
                        $networkName = 'default['.($defaultName).']';
                    }

                    //$this->networkName[] = $networkName; // --net 은 배열이 아니다.
                    $this->networkName = [$networkName];

                    if (true === isset($dockerNetworks[$networkName])) {
                        $networkRmcommand = [
                            'docker',
                            'network',
                            'rm',
                            $networkName,
                            '2>&1',
                        ];
                        $this->process($networkRmcommand, ['print' => false]);
                        unset($dockerNetworks[$networkName]);
                    }

                    foreach ($dockerNetworks as $dockerNetworkName => $dockerNetworkSubnet) {
                        foreach ($network['ipam']['config'] as $configSubnet) {
                            if (true === isset($configSubnet['subnet']) && $configSubnet['subnet'] == $dockerNetworkSubnet) {
                                $this->message(\Peanut\Console\Color::gettext($networkName.' conflicts with network '.$dockerNetworkName.', subnet '.$dockerNetworkSubnet, 'red'));
                                echo 'delete? [y/N]: ';
                                $handle = fopen('php://stdin', 'r');
                                $line   = fgets($handle);
                                fclose($handle);

                                if (false === in_array(trim($line), ['y', 'Y'])) {
                                    throw new \Peanut\Console\Exception($networkName.' conflicts with network '.$dockerNetworkName.', subnet '.$dockerNetworkSubnet);
                                }
                                $networkRmCommand = [
                                                        'docker',
                                        'network',
                                        'rm',
                                        $dockerNetworkName,
                                    ];
                                $this->process($networkRmCommand, ['print' => false]);
                            }
                        }
                    }

                    $subnet = [];

                    foreach ($network['ipam']['config'] as $configSubnet) {
                        if (true === isset($configSubnet['subnet']) && $configSubnet['subnet']) {
                            $subnet[] = $configSubnet['subnet'];
                        }
                    }

                    $networkCreateCommand = [
                        'docker',
                        'network',
                        'create',
                        '--driver=bridge',
                    ];

                    if ($subnet) {
                        $networkCreateCommand[] = '--subnet='.implode(' --subnet=', $subnet);
                    } else {
                        
                        $subnetFile = $this->process('echo $HOME', ['print' => false])->toString().'/.docker/docker-machine-subnet.yaml';

                        if (false === is_file($subnetFile)) {
                            mkdir($this->process('echo $HOME', ['print' => false])->toString().'/.docker/');
                            $this->process('touch '.$subnetFile, ['print' => false]);
                        }

                        $subnets = \App\Helpers\Yaml::parseFile($subnetFile);

                        if (false === is_array($subnets)) {
                            $subnets = [];
                        }

                        if (true === isset($subnets[$machineName][$projectName])) {
                            $subnet = $subnets[$machineName][$projectName];
                        } else {
                            $bridge = $this->process([
                                'docker',
                                'network',
                                'inspect',
                                "--format='{{range .IPAM.Config}}{{.Subnet}}{{end}}'",
                                'bridge',
                            ], ['print' => false])->toString();

                            $subnetIps = [];
                            foreach ($subnets as $machines) {
                                if (true === is_array($machines)) {
                                    foreach ($machines as $projectIp) {
                                        $subnetIps[] = $projectIp;
                                    }
                                }
                            }

                            while (1) {
                                $subnet = '172.'.rand(0, 255).'.0.0/16';
                                //$subnet = '172.17.0.0/16';

                                if ($subnet == $bridge) {
                                    continue;
                                }

                                if (false == in_array($subnet, $subnetIps)) {
                                    break;
                                }
                            }
                            $subnets[$machineName][$projectName] = $subnet;
                            \App\Helpers\Yaml::dumpFile($subnetFile, $subnets);
                        }
                        $networkCreateCommand[] = '--subnet='.$subnet;
                    }

                    $networkCreateCommand[] = $networkName;
                    $networkCreateCommand[] = '2>&1';
                    $this->process($networkCreateCommand, ['print' => false]);

                    $networkInspectCommand = [
                        'docker',
                        'network',
                        'inspect',
                        '--format="{{range .IPAM.Config}}{{.Subnet}}{{end}}"',
                        $networkName,
                        '2>&1',
                    ];
                    $subnet = $this->process($networkInspectCommand, ['print' => false])->toArray();

                    if (!$subnet) {
                        throw new \Peanut\Console\Exception('network '.$networkName.' not found');
                    }

                    echo \Peanut\Console\Color::gettext('network | ', 'white').'recreate '.$networkName.', subnet '.implode(' ', $subnet).PHP_EOL;
                }
            }
        }

        // create or run
        {
            if ('attach' == $mode) {
                echo \Peanut\Console\Color::gettext('start   | ', 'white');
            } else {
                echo \Peanut\Console\Color::gettext('run     | ', 'white');
            }

            $runCommands = [];

            foreach ($compose['services'] as $serviceName => $service) {
                $command = [];

                echo $service['org_name'];

                if ('attach' == $mode) {
                    $command[] = 'docker create';
                    $command[] = '-a STDIN';
                    $command[] = '-a STDOUT';
                    $command[] = '-a STDERR';
                    $command[] = '-i';
                } else {
                    $command[] = 'docker run';
                    $command[] = '-d';
                    $command[] = '-i';

                    if (true === isset($service['tty'])) {
                        $command[] = '--tty';
                    }
                }

                if (true === isset($service['privileged'])) {
                    $command[] = '--privileged';
                }

                if ($this->networkName) {
                    foreach ($this->networkName as $networkName) {
                        $command[] = '--net='.$networkName;
                    }
                }

                if (true === isset($service['networks'])) {
                    foreach ($service['networks'] as $name => $conf) {
                        if (true === isset($conf['ipv4_address'])) {
                            $command[] = '--ip='.$conf['ipv4_address'];
                        }

                        if (true === isset($conf['ipv6_address'])) {
                            $command[] = '--ip6='.$conf['ipv6_address'];
                        }
                    }
                }

                if (true === isset($service['dns'])) {
                    foreach ($service['dns'] as $value) {
                        $command[] = '--dns='.$value;
                    }
                }

                if (true === isset($service['env-file'])) {
                    foreach ($service['env-file'] as $value) {
                        $command[] = '--env-file='.$value;
                    }
                }

                $command[] = '-e TERM=xterm';

                if (true === isset($service['environment'])) {
                    foreach ($service['environment'] as $key => $value) {
                        if (true === is_array($value)) {
                            $value = "'".json_encode($value, JSON_UNESCAPED_SLASHES)."'";
                        }
                        $command[] = '-e '.$key.'='.escapeshellarg($value);
                    }
                }

                if (true === isset($service['logging'])) {
                    if (true === isset($service['logging']['driver'])) {
                        $command[] = '--log-driver '.$service['logging']['driver'];
                    }
                    if (true === is_array($service['logging']['options'])) {
                        foreach ($service['logging']['options'] as $optionKey => $optionValue) {
                            $command[] = '--log-opt '.$optionKey.'='.$optionValue;
                        }
                    }
                }

                if (true === isset($service['expose'])) {
                    foreach ($service['expose'] as $key => $value) {
                        $command[] = '--expose='.$value;
                    }
                }

                if (true === isset($service['user'])) {
                    $command[] = '--user='.$service['user'];
                }

                if (true === isset($service['hostname'])) {
                    $command[] = '--hostname="'.$service['hostname'].'"';
                }

                $addHost = [];

                if (true === isset($service['links'])) {
                    foreach ($service['links'] as $key => $value) {
                        foreach ($this->networkName as $networkName) {
                            $inspectCommand = [
                                'docker',
                                'inspect',
                                '--format="service={{index .Config.Labels \"com.docker.bootapp.service\"}}&&ip={{with index .NetworkSettings.Networks \"'.$networkName.'\"}}{{.IPAddress}}{{end}}"',
                                $this->getContainerName($value),
                            ];

                            $str = $this->process($inspectCommand, ['print' => false])->toString();
                            parse_str($str, $out);

                            if (true === isset($out['ip']) && true === isset($out['service'])) {
                                $ip           = $out['ip'];
                                $serviceName  = $out['service'];
                                $addHost[$ip] = $serviceName.' '.$this->getContainerName($value);
                            }
                        }

                        $command[] = '--link='.$this->getContainerName($value);
                    }
                }

                foreach ($addHost as $ip => $host) {
                    $command[] = '--add-host="'.$host.'":'.$ip;
                }

                if (true === isset($service['extra_hosts'])) {
                    foreach ($service['extra_hosts'] as $value) {
                        $command[] = '--add-host="'.$value.'"';
                    }
                }
                //$command[] = '-P';

                if (true === isset($service['net'])) {
                    $command[] = '--net='.$service['net'];
                }

                if (true === isset($service['working_dir'])) {
                    $command[] = '--workdir='.$service['working_dir'];
                }

                if (true === isset($service['ports'])) {
                    foreach ($service['ports'] as $value) {
                        $command[] = '--publish='.$value;
                    }
                }
                //$command[] = '-P';

                if (true === isset($service['restart'])) {
                    $command[] = '--restart='.$service['restart'];
                }

                if (true === isset($service['volumes'])) {
                    foreach ($service['volumes'] as $value) {
                        $command[] = '-v '.$this->volumeRealPath($value);
                    }
                }

                if (true === isset($service['volumes_from'])) {
                    foreach ($service['volumes_from'] as $value) {
                        $command[] = '--volumes-from='.($projectName ? $projectName.'-'.$value : $value);
                    }
                }

                if (true === isset($service['entrypoint'])) {
                    $command[] = '--entrypoint='.$service['entrypoint'];
                }

                $command[] = '--name='.$service['name'];

                if (true === isset($service['labels'])) {
                    foreach ($service['labels'] as $labelName => $labelValue) {
                        $command[] = '--label '.$labelName.'="'.$labelValue.'"';
                    }
                }

                if (true === isset($service['environment']['DOMAIN'])) {
                    //if (false === strpos($service['environment']['DOMAIN'], ' ')) {
                    $command[] = '--label com.docker.bootapp.domain="'.$service['environment']['DOMAIN'].'"';
                    //} else {
                    //    throw new \Peanut\Console\Exception('domain name not valid');
                    //}
                }

                if (true === isset($service['image'])) {
                    $command[] = $service['image'];
                }

                if (true === isset($service['build'])) {
                    $command[] = $service['name'];
                }

                if (true === isset($service['command'])) {
                    $command[] = $service['command'];
                }

                $runCommands[] = implode(' ', $command);
                file_put_contents($this->getcwd().'/.bootapp.log', date('Y-m-d H:i:s').PHP_EOL.implode(' ', $command).PHP_EOL.PHP_EOL, FILE_APPEND);

                $this->process($command, ['print' => '.']); // create

                if ('attach' == $mode) {
                    $command = [
                        'docker',
                        'start',
                        $service['name'],
                    ];
                    $this->process($command, ['print' => false]);

                    $command = [
                        'docker',
                        'logs',
                        $service['name'],
                        '&&',
                        'docker',
                        'attach',
                        '--sig-proxy=true',
                        $service['name'],
                    ];
                    $this->childProcess($service['name'], implode(' ', $command));
                }

                echo ' ';
            }

            echo PHP_EOL;
        }
    }
}
