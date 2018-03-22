<?php
namespace App\Helpers;

use Symfony\Component\Yaml\Yaml as SymfonyYaml;

class Yaml
{
    /**
     * @param  $fileName
     * @return mixed
     */
    public static function parseFile($fileName)
    {
        $folder      = '';
        $paths       = explode('/', trim(getcwd(), '/'));
        $fixFileName = $fileName;
        for ($i=0,$j=count($paths);$i < $j;$i++) {
            $fixFileName = $folder.$fixFileName;

            if (true === file_exists($fixFileName)) {
                try {
                    return static::parse(file_get_contents($fixFileName));
                } catch (\Exception $e) {
                    throw new \Peanut\Console\Exception($e);
                }
                break;
            }
            $folder = '../';
        }

        throw new \Peanut\Console\Exception('"'.$fileName.'" file not exists.');
    }

    /**
     * @param  string  $yaml
     * @return array
     */
    public static function parse($yaml)
    {
        try {
            $array = SymfonyYaml::parse($yaml);
        } catch (\Exception $e) {
            throw new \Peanut\Console\Exception($e);
        }

        return $array;
    }

    /**
     * @param  $fileName
     * @param  $array
     * @return bool
     */
    public static function dumpFile($fileName, $array)
    {
        //static::message('yaml dump file '.$fileName);

        try {
            return file_put_contents($fileName, static::dump($array));
        } catch (\Exception $e) {
            throw new \Peanut\Console\Exception($e);
        }
    }

    /**
     * @param  array  $array
     * @return yaml
     */
    public static function dump($array)
    {
        try {
            return trim(SymfonyYaml::dump($array, 10, 2));
        } catch (\Exception $e) {
            throw new \Peanut\Console\Exception($e);
        }
    }
}
