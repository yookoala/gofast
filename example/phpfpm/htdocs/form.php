<?php

$submitted = !empty($_POST) json_encode($_POST) : FALSE;

?>
<?php if ($submitted != FALSE): ?>
<?php print $submitted; ?>
<?php else: ?>
<!DOCTYPE html>
<html>
<head>
  <title>Simple Form</title>
</head>
<body>
  <h1>Simple Form</h1>
  <form action="./form.php" method="POST">
    <label for="text_input">Text Input</label>
    <input type="text" value="text_input" />
    <button type="submit">Submit</button>
  </form>
</body>
</html>
<?php endif; ?>
