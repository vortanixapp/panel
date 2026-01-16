from aiohttp import web

from vtxdaemon.config import DAEMON_TOKEN


@web.middleware
async def auth_middleware(request, handler):
    if request.method == 'OPTIONS' or request.path == '/health':
        return await handler(request)

    token = request.headers.get('X-Location-Daemon-Token', '')
    if DAEMON_TOKEN and token != DAEMON_TOKEN:
        return web.json_response({'error': 'Unauthorized'}, status=401)

    return await handler(request)


@web.middleware
async def cors_middleware(request, handler):
    if request.method == 'OPTIONS':
        response = web.Response(status=204)
    else:
        response = await handler(request)

    response.headers['Access-Control-Allow-Origin'] = '*'
    response.headers['Access-Control-Allow-Methods'] = 'GET,POST,OPTIONS'
    response.headers['Access-Control-Allow-Headers'] = 'Content-Type,X-Location-Daemon-Token'
    return response
