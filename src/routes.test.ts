
import { test, expect } from 'bun:test'

import { healthCheck, echo } from './routes'

const PORT = 3000;

test("test routes", async () => {
    const server = Bun.serve({
        development: true,
        port: PORT,
        routes: {
            '/health': healthCheck,
            '/': echo
        }
    })

    var request = new Request(`${server.url}/health`)
    var response = await fetch(request)
    expect(response.ok).toBe(true)

    request = new Request(`${server.url}`)
    response = await fetch(request)
    expect(response.ok).toBe(true)

    await server.stop(true)
})
