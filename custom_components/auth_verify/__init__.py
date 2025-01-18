from homeassistant.core import HomeAssistant
from homeassistant.config_entries import ConfigEntry
from homeassistant.helpers.typing import ConfigType
from .auth_verify import AuthVerify

DOMAIN = "auth_verify" 

async def async_setup(hass: HomeAssistant, config: dict):
    auth_verify_config = config.get(DOMAIN, [])
    
    hass.http.register_view(AuthVerify(hass, auth_verify_config))
    
    return True