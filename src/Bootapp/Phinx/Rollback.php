<?php
namespace Bootapp\Phinx;

use Bootapp\Phinx;
use Symfony\Component\Console\Input\InputOption;

class Rollback extends \Phinx\Console\Command\Rollback
{
    use Phinx;
    /**
     * {@inheritdoc}
     */
    public function configure()
    {
        $this->addOption('--environment', '-e', InputOption::VALUE_REQUIRED, 'The target environment', $this->defaultEnvironment);

        $this->setName('phinx:rollback')
            ->setDescription('Rollback the last or to a specific migration')
            ->addOption('--target', '-t', InputOption::VALUE_REQUIRED, 'The version number to rollback to')
            ->addOption('--date', '-d', InputOption::VALUE_REQUIRED, 'The date to rollback to')
            ->setHelp(
                <<<EOT
The <info>rollback</info> command reverts the last migration, or optionally up to a specific version

<info>bootapp phinx:rollback -e development</info>
<info>bootapp phinx:rollback -e development -t 20111018185412</info>
<info>bootapp phinx:rollback -e development -d 20111018</info>
<info>bootapp phinx:rollback -e development -v</info>

EOT
            );
    }
}
