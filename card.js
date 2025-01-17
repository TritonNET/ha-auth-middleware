// Download latest from https://cdn.jsdelivr.net/gh/lit/dist@3.1.4/core/lit-core.min.js
import { LitElement, html, css, } from "https://cdn.jsdelivr.net/gh/lit/dist@3.1.4/core/lit-core.min.js";

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