var fcgi = require('node-fastcgi');

var listenTarget = process.env.TEST_FCGI_SOCK || './node-fastcgi.sock';


responder = function(req, res) {
  if (req.method === 'GET') {
    res.writeHead(200, { 'Content-Type': 'text/plain' });
    var body = 'hello index';
    const custom = []; // custom message

    // search for header messages
    for (headerKey in req.headers) {
      if (headerKey.match(/^my-/)) {
        let key = /^my-(.+?)-message/.exec(headerKey)[1];
        custom.push(`${key}: ${req.headers[headerKey]}`);
      }
    }

    if (custom.length > 0) {
      res.end(custom.join("\n") + `\n${body}`);
      return;
    }
    res.end(body);
  } else if (req.method === 'POST') {
    res.writeHead(200, { 'Content-Type': 'text/plain' });
    var body = '';

    req.on('data', function(data) { body += data.toString(); });
    req.on('end', function() {
      res.end('Received data:\n' + body);
    });
  } else {
    res.writeHead(501);
    res.end();
  }
}
authorizer = function (req, res) {
  if (req.method === 'GET' || req.method === 'POST') {
    // simple authorizer check
    //
    // Note: according to spec, authorizer have no
    // information from:
    // - CONTENT_LENGTH;
    // - PATH_INFO;
    // - PATH_TRANSLATED; and
    // - SCRIPT_NAME
    //
    req.on('complete', function () {
      const authHeader = req.headers.authorization || '';
      if (authHeader === 'hello-auth') {
        res.setVariable('My-Hello-Message', 'howdy!')
        res.setVariable('My-Foo-Message', 'bar!')
        res.writeHead(200);
        res.end();
        return;
      }
      res.writeHead(403, { 'Content-Type': 'text/plain' });
      res.end('authorizer app: permission denied');
    });
  } else {
    res.writeHead(501, { 'Content-Type': 'text/plain' });
    res.end('unsupported method');
  }
}
filter = function (req, res) {
  // a simple filter to reverse the data string as
  // response body
  if (req.method === 'GET' || req.method === 'POST') {
    res.writeHead(200, { 'Content-Type': 'text/plain' });
    var body = '';
    var toFilter = '';

    // read post data, if any
    req.on('data', function(data) { body += data.toString(); });
    res.on('end', function () {
      res.end('error: no data to filter with')
    });

    // read on data stream until stream fulfilled
    req.socket.dataStream.on('data', function (data) {
      // the filter logic for test
      // Note: you may do eval(data) if you want to run data as js
      toFilter += data.toString();
    });
    req.socket.dataStream.on('end', function (data) {
      // the filter logic for test
      // Note: you may do eval(data) if you want to run data as js
      res.end(toFilter.split('').reverse().join(''));
    });

  } else {
    res.writeHead(501);
    res.end();
  }
}

/**
 * function createServer([responder], [authorizer], [filter], [config])
 * Creates and returns a FastCGI server object. Compatible with http.createServer
 *
 * Arguments:
 *   - responder (optional): callback for FastCGI responder requests (normal HTTP requests, 'request' event)
 *   - authorizer (optional): callback for FastCGI authorizer requests ('authorize' event)
 *   - filter (optional): callback for FastCGI filter requests ('filter' event)
 *   - config (optional): server configuration (default: { maxConns: 2000, maxReqs: 2000, multiplex: true, valueMap: {} })
 */
fcgi.createServer(responder, authorizer, filter).listen(listenTarget);
