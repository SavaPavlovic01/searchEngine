import http.server
import socketserver

PORT = 8000
DIRECTORY = "."

Handler = http.server.SimpleHTTPRequestHandler

# Optional: change working directory
import os
os.chdir(DIRECTORY)

with socketserver.TCPServer(("", PORT), Handler) as httpd:
    print(f"Serving at http://localhost:{PORT}")
    httpd.serve_forever()