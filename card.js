// Download latest from https://cdn.jsdelivr.net/gh/lit/dist@3.1.4/core/lit-core.min.js
import { LitElement, html, css } from "https://cdn.jsdelivr.net/gh/lit/dist@3.1.4/core/lit-core.min.js";

class AuthWebpageCard extends LitElement {

    static get properties() {
        return {
            hass: undefined,
            config: undefined,
            url: undefined,
        };
    }

    constructor() {
        // always call super() first
        super();

        this.url = "";
    }

    setConfig(config) {
        this.config = config;

        if (this.config.config != undefined) {
            const type = this.config.config.type;
            switch (type) {
                case "generic":
                    this.url = this.config.config.url;
                    break;
            }
        }
        
        this.setIframeCookie();
    }

    setIframeCookie() {
        try {
            const hassToken = localStorage.getItem('hassTokens');
            if (hassToken == null) {
                console.error("No hass token found in local storage");
                return;
            }

            const hassTokenJson = JSON.parse(hassToken);

            const accessToken = hassTokenJson.access_token;
            if (!accessToken) {
                console.error("No access token found in hass token");
                return;
            }

            const expiresIn = hassTokenJson.expires_in; // Token expiration in seconds
            if (!expiresIn) {
                console.error("No expiration time found in hass token");
                return;
            }

            // Calculate the expiration time in milliseconds
            const expiresInMs = expiresIn * 1000; // Convert seconds to milliseconds
            const expiresAt = new Date(Date.now() + expiresInMs).toUTCString();

            // Set the cookie
            document.cookie = `haatc=${accessToken}; path=/; domain=.${location.hostname}; expires=${expiresAt}; Secure; SameSite=None`;
            console.log(`Cookie set with expiration at ${expiresAt}`);

            // Schedule the cookie refresh based on the expires_in value
            setTimeout(() => {
                console.log("Refreshing cookie...");
                this.setIframeCookie(); // Re-read the latest token from localStorage
            }, expiresInMs - 500); // Refresh 500ms before it expires for safety
        } catch (error) {
            console.error("Error setting iframe cookie:", error);
        }
    }

    
    render() {
        return html`
              <iframe class="chart-frame" src="${this.url}"></iframe>
            `;
    }

    static get styles() {
        return css`
          .chart-frame {
            border: none; 
            margin: 0; 
            padding: 0;
          }      
        `;
    }
}

customElements.define('auth-webpage', AuthWebpageCard);
