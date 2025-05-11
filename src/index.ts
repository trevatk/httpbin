
import { healthCheck, echo } from './routes'

const server = Bun.serve({
    routes: {
        "/health": healthCheck,
        "/": echo
    },
    error: (error) => {
        console.error(`internal server error ${error.message}`)
    }
})

console.log(`http/1 server listening at: ${server.url}`);

process.on('SIGINT', async () => {
    console.log(`received SIGINT signal shutting down http/1 server`)

    // close active connections
    const forceClose = true;

    await server.stop(forceClose)
})

export default {
    server
}
