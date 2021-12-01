import http.server
import socketserver

PORT = 8000

Handler = http.server.SimpleHTTPRequestHandler
Handler.extensions_map.update(
    {".wasm": "application/wasm", ".js": "application/javascript"}
)

socketserver.TCPServer.allow_reuse_address = True
with socketserver.TCPServer(("", PORT), Handler) as httpd:
    httpd.allow_reuse_address = True
    print(f"🎅 http://localhost:{PORT}")
    httpd.serve_forever()
