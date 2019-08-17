<?php
namespace App\Traits\Docker;

trait Compose
{
    public function fileGenerate()
    {
        $config = $this->getConfig();

        if (true === isset($config['stages'])) {
            $stages = $config['stages'];
        } else {
            $stages = [
                'local' => [
                    'services' => [],
                ],
            ];
        }

        foreach ($stages as $stageName => $stage) {
            $compose = [
                'version' => '2',
            ];

            if (true === isset($config['volumes']) || true === isset($stage['volumes'])) {
                if (false === isset($config['volumes'])) {
                    $config['volumes'] = [];
                }

                if (false === isset($stage['volumes'])) {
                    $stage['volumes'] = [];
                }

                $compose['volumes'] = array_merge_recursive($config['volumes'], $stage['volumes']);
            }

            if (true === isset($config['services']) || true === isset($stage['services'])) {
                if (false === isset($config['services'])) {
                    $config['services'] = [];
                }

                if (false === isset($stage['services'])) {
                    $stage['services'] = [];
                }

                $services = array_merge_recursive($config['services'], $stage['services']);

                $links = [];

                foreach ($services as $key => $value) {
                    if (true === isset($value['links'])) {
                        foreach ($value['links'] as $link) {
                            if (false !== strpos($link, ':')) {
                                $links[$key][] = explode(':', $link)[0];
                            } else {
                                $links[$key][] = $link;
                            }
                        }
                    } else {
                        $links[$key] = [];
                    }
                }

                try {
                    $result = \App\Helpers\Dependency::sort($links);
                } catch (\Exception $e) {
                    throw new \Peanut\Console\Exception($e);
                }

                $compose['services'] = [];

                foreach ($result as $serviceName) {
                    if (false === isset($services[$serviceName]['labels'])) {
                        $services[$serviceName]['labels'] = [];
                    }

                    $services[$serviceName]['labels'] += [
                        'com.docker.bootapp.service' => $serviceName,
                        'com.docker.bootapp.name'    => $this->getContainerName($serviceName),
                        'com.docker.bootapp.project' => $this->getProjectName(),
                    ];

                    if (true === isset($services[$serviceName]['environment']['DOMAIN'])) {
                        $services[$serviceName]['labels']['com.docker.bootapp.domain'] = $services[$serviceName]['environment']['DOMAIN'];

                        //if (false === strpos($services[$serviceName]['environment']['DOMAIN'], ' ')) {
                        $services[$serviceName]['labels']['com.docker.bootapp.domain'] = $services[$serviceName]['environment']['DOMAIN'];
                        //} else {
                        //    throw new \Peanut\Console\Exception('domain name not valid');
                        //}
                    }

                    if (true === isset($services[$serviceName]['container_name'])) {
                        $services[$serviceName]['container_name'] = $this->getContainerName($serviceName);
                    }

                    $compose['services'][$serviceName] = $services[$serviceName];
                }
            }

            if (true === isset($config['networks']) || true === isset($stage['networks'])) {
                if (false === isset($config['networks'])) {
                    $config['networks'] = [];
                }

                if (false === isset($stage['networks'])) {
                    $stage['networks'] = [];
                }

                $compose['networks'] = array_merge_recursive($config['networks'], $stage['networks']);
            }

            // custom key, environment_from 처리
            {
                foreach ($compose['services'] as $service_name => $service) {
                    if (true === isset($service['environment_from'])) {
                        foreach ($service['environment_from'] as $from_name => $from) {
                            foreach ($from as $env_name) {
                                $env_alias = preg_split('/:/D', $env_name);

                                if (true === isset($env_alias[1])) {
                                    $compose['services'][$service_name]['environment'][$env_alias[1]] = $compose['services'][$from_name]['environment'][$env_alias[0]];
                                } else {
                                    $compose['services'][$service_name]['environment'][$name] = $compose['services'][$env_alias[0]]['environment'][$env_alias[0]];
                                }
                            }
                        }

                        unset($compose['services'][$service_name]['environment_from']);
                    }
                }
            }

            {

                foreach ($compose['services'] as $service_name => &$service) {
                    if (true === isset($service['environment']) && $service['environment']) {
                        foreach ($service['environment'] as $key => &$value) {
                            if ($key == 0 && is_null($value)) {
                                unset($service['environment'][$key]);
                            } elseif (true === is_array($value)) {
                                $value = "'".json_encode($value, JSON_UNESCAPED_SLASHES)."'";
                            }
                        }
                    } else {
                        unset($service['environment']);
                    }
                }
            }

            \App\Helpers\Yaml::dumpFile($this->getcwd().'/docker-compose.'.$stageName.'.yml', $compose);
        }
    }
}
