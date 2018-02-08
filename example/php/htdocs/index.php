<?php

/**
 * File for testing gofast ability to handler
 * php-fpm application.
 *
 * @category File
 */

// some dummy header to test with
header("X-Hello: World");
header("X-Foo: Bar");

// dummy content
echo "hello index";
