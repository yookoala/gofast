var fcgi = require('node-fastcgi');

var listenTarget = process.env.TEST_FCGI_SOCK || './node-fastcgi.sock';

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
fcgi.createServer(function(req, res) {
  if (req.method === 'GET') {
    res.writeHead(200, { 'Content-Type': 'text/plain' });
    res.end("hello index");
  } else if (req.method === 'POST') {
    res.writeHead(200, { 'Content-Type': 'text/plain' });
    var body = "";

    req.on('data', function(data) { body += data.toString(); });
    req.on('end', function() {
      res.end("Received data:\n" + body);
    });
  } else {
    res.writeHead(501);
    res.end();
  }
}).listen(listenTarget);