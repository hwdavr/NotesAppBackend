# Notion account linking

Authenticated clients can start a Notion OAuth authorization, inspect their connection, and disconnect it. The public callback consumes a short-lived single-use state, exchanges the code with Notion, stores the access token encrypted, and returns to the mobile app. Disconnect invalidates pending and in-flight authorizations. Configure the Notion client and encryption key as described in the README. Notion page and folder operations are not part of this delivery.

The feature was verified with focused unit and authenticated router tests, OpenAPI schema and response checks, isolated PostgreSQL migration and transaction tests, full integration regression, and `make check`. See the dated feature workspace for evidence and limitations.
