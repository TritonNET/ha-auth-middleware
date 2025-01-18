import logging
from aiohttp import web
from homeassistant.components.http.view import HomeAssistantView
from homeassistant.core import HomeAssistant

# Initialize a logger for this component
_LOGGER = logging.getLogger(__name__)


class AuthVerify(HomeAssistantView):
    url = "/api/auth_verify/{module}"  # Dynamic route with a path parameter
    name = "api:auth_verify"
    requires_auth = True

    def __init__(self, hass: HomeAssistant, config: dict) -> None:
        self.hass = hass
        self.config = config  # Store the auth_verify configuration

    async def get(self, request: web.Request, module: str):
        # Look up the email for the module in the configuration
        email = next((item["email"] for item in self.config if item["module"] == module), None)
        if not email:
            return web.json_response({"error": f"module '{module}' not found"}, status=404)

        # Return the email as the response
        return web.json_response({"email": email}, status=200)
