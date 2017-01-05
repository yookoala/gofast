#!/usr/bin/env python3
# -*- coding: UTF-8 -*-

import sys, os, logging
from html import escape
from flup.server.fcgi import WSGIServer

# example app
def app(environ, start_response):
    logging.info("triggered app")
    start_response('200 OK', [('Content-Type', 'text/html')])
    yield "hello index"
    #yield '<h1>FastCGI Environment</h1>'
    #yield '<table>'
    #for k, v in sorted(environ.items()):
    #     yield "<tr><th>{0}</th><td>{1}</td></tr>\n".format(
    #         escape(k), escape(v))
    #yield '</table>'

# process to gracefully run example app as a fastcgi application
def main():

    # setup logging
    logging.basicConfig(
        datefmt='%Y-%m-%d %H:%M:%S',
        format='%(asctime)s [webapp.py] %(levelname)s: %(message)s',
        level=logging.INFO)

    # check if socket defined
    if 'TEST_PY3FCGI_SOCK' not in os.environ:
        logging.error('env TEST_PY3FCGI_SOCK not found')
        exit(1)
    socket = os.environ['TEST_PY3FCGI_SOCK']
    logging.info("listening socket: " + socket)

    # run the server
    try:
        WSGIServer(app, bindAddress=socket, umask=0000).run()
    except (KeyboardInterrupt, SystemExit, SystemError):
        logging.info("Shutdown requested...exiting")
    except Exception:
        traceback.print_exc(file=sys.stdout)
    finally:
        if os.path.exists(socket):
            logging.info("removing socket: " + socket)
            os.remove(socket)
        else:
            logging.info("socket not exists: " + socket)

    # exit gracefully
    logging.info("bye bye")
    sys.exit(0)

if __name__ == '__main__':
    main()
