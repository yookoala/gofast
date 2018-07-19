<?php

// flag for showing only error
if (isset($_GET['error_only']) && $_GET['error_only']) {
  error_log('only error.');
  exit;
}

// some output
echo "1. some standard output.\n";

// output to error stream
error_log("2. some error.");

// standard output after error stream
echo "3. some more standard output.\n";

// output to error stream
error_log("4. some more error.");

?>
5. unparsed.
