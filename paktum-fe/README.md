# paktum-fe

This is the basic frontend for the Paktum server.

It accesses the API using graphql, and will by default connect to /graphql on the same origin as it's intended to be bundled along with Paktum itself.

To change this behaviour, edit the `src/API.ts` file and set the endpoint accordingly.