<?php
namespace App\Helpers;

class Dependency
{
    /**
     * @var array
     */
    protected $elements = [];

    /**
     * @var mixed
     */
    protected $sorted;

    /**
     * @var int
     */
    protected $position = 0;

    /**
     * @param $array
     */
    public function __construct($array)
    {
    }

    /**
     * @return mixed
     * @param null|mixed $array
     */
    public static function sort($array = null)
    {
        $instance = new Self($array);

        if (true === is_array($array)) {
            $instance->set($array);
        }

        return $instance->doSort()->toArray();
    }

    /**
     * @param array $elements
     */
    public function set(array $elements)
    {
        foreach ($elements as $element => $dependencies) {
            $this->add($element, $dependencies);
        }
    }

    /**
     * @param $element
     * @param array      $dependencies
     */
    public function add($element, $dependencies = [])
    {
        // Add
        $this->elements[$element] = (object) [
            'id'           => $element,
            'dependencies' => (array) $dependencies,
            'visited'      => false,
        ];
    }

    /**
     * @param $element
     * @param $parents
     */
    public function visit($element, &$parents = null)
    {
        if (!$element->visited) {
            $parents[$element->id] = true;
            $element->visited      = true;

            foreach ($element->dependencies as $dependency) {
                if (isset($this->elements[$dependency])) {
                    $newParents = $parents;
                    $this->visit($this->elements[$dependency], $newParents);
                } else {
                    //print_r([$element->id, $dependency]);
                    throw new \Peanut\Console\Exception($element->id.' dependencies '.$dependency.' not found');
                }
            }

            $this->addToList($element);
        }
    }

    /**
     * @return mixed
     */
    public function doSort()
    {
        try {
            $this->sorted = new \SplFixedArray(count($this->elements));
        } catch (\Exception $e) {
            throw new \Peanut\Concole\Exception($e);
        }

        foreach ($this->elements as $element) {
            $parents = [];
            $this->visit($element, $parents);
        }

        return $this->sorted;
    }

    /**
     * @param $element
     */
    protected function addToList($element)
    {
        $this->sorted[$this->position++] = $element->id;
    }
}
