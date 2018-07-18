<?php

if (!empty($_POST)) {
  $method = "POST";
  $submitted = !empty($_POST) ? var_export($_POST, TRUE) : FALSE;
} else if (!empty($_GET)) {
  $method = "GET";
  $submitted = !empty($_GET) ? var_export($_GET, TRUE) : FALSE;
}

$entityBody = file_get_contents('php://input');

?>
<?php if ($submitted != FALSE): ?>
<?php print '$_' . $method . ' = ' . $submitted; ?>
<?php else: ?>
<!DOCTYPE html>
<html>
<head>
  <title>Simple Form</title>
</head>
<body>

<textarea>
file_get_contents("php://input") = <?php var_export($entityBody); ?>;

$HTTP_RAW_POST_DATA = <?php var_export(file_get_contents('php://input')); ?>;

$_SERVER = <?php var_export($_SERVER); ?>;

$_REQUEST = <?php var_export($_REQUEST); ?>;

$_GET = <?php var_export($_GET); ?>;

$_POST = <?php var_export($_POST); ?>;

</textarea>

  <h1>Simple Form</h1>
  <form action="./form.php" method="POST">
    <label for="text_input">Text Input</label>
    <input type="text" value="text_input" />
    <button type="submit">Submit</button>
  </form>
</body>
</html>
<?php endif; ?>
