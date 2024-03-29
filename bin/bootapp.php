#!/usr/bin/env php
<?php
date_default_timezone_set('Asia/Seoul');
error_reporting(-1);

$loaded = false;

foreach ([__DIR__.'/../../../autoload.php', __DIR__.'/../vendor/autoload.php'] as $file) {
    if (file_exists($file)) {
        require $file;
        $loaded = true;
        break;
    }
}

if (!$loaded) {
    die(
        'You need to set up the project dependencies using the following commands:'.PHP_EOL.
        'wget http://getcomposer.org/composer.phar'.PHP_EOL.
        'php composer.phar install'.PHP_EOL
    );
}

echo <<<LOGO
 _                 _
| |__   ___   ___ | |_    ____ ____  ____
|  _ \ / _ \ / _ \| __|  / _  |  _ \|  _ \
| |_) | (_) | (_) | |_  | (_| | |_) | |_) |
|_.__/ \___/ \___/ \__|  \__,_| .__/| .__/  0.0.0
                              |_|   |_|
LOGO;

set_error_handler(function ($errno, $errstr, $errfile, $errline) {
    // if (!(error_reporting() & $errno)) {
    //     // This error code is not included in error_reporting, so let it fall
    //     // through to the standard PHP error handler
    //     return false;
    // }

    switch ($errno) {
        case E_USER_ERROR:
        case E_USER_WARNING:
        case E_USER_NOTICE:
            throw new \Peanut\Console\Exception("[{$errno}] {$errstr} on {$errfile}:{$errline}");
            break;
        default:
            echo "Unknown error type: [{$errno}] $errstr on {$errfile}:{$errline}\n";
            break;
    }

    /* Don't execute PHP internal error handler */
    return true;
});
try {
    $app = new \Peanut\Console\Application('Bootapp', '0.0.0');

    $app->option('verbose', ['require' => false, 'alias' => 'v|vv|vvv', 'value' => false]);
    $app->option('no-update', ['require' => false, 'alias' => 'n', 'value' => false]);

    $app->command(new \App\Controllers\Selfupdate());
    $app->command(new \App\Controllers\Up());
    $app->command(new \App\Controllers\Halt());

    $app->command(new \App\Controllers\Ls());
    $app->command(new \App\Controllers\Rm());

    $app->command(new \App\Controllers\Fix());
    $app->command(new \App\Controllers\Log());
    $app->command(new \App\Controllers\Inspect());
    $app->command(new \App\Controllers\Ssh());

    $app->command(new \App\Controllers\Task());
    $app->command(new \App\Controllers\Env());

    $app->command(new \App\Controllers\Pause());
    $app->command(new \App\Controllers\Unpause());

    $app->match();
} catch (\Peanut\Console\Exception $e) {
    print_r($e->getMessage().' in '.$e->getFile().' on '.$e->getLine().PHP_EOL);
}