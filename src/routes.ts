
export async function healthCheck(_req: Bun.BunRequest) {
    return new Response('OK', { status: 200})
}

export async function echo(_req: Bun.BunRequest) {
    return new Response('hello world', { status: 200 })
}