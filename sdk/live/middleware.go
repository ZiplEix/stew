package live

import (
	"net/http"
	"os"
)

const liveReloadScript = `  
<button id="stew-reload-trigger" 
        hx-get="/" 
        hx-target="body" 
        hx-select="body" 
        hx-swap="morph" 
        style="display:none">
</button>

<script>
  let isChecking = false;

  function connect() {
    const ev = new EventSource('/stew/live');
    
    ev.onmessage = (event) => {
      if (event.data === 'reload') {
        doMorph();
      }
    };

    ev.onerror = () => {
      ev.close();
      if (!isChecking) {
        isChecking = true;
        checkServerAndMorph();
      }
    };
  }

  async function checkServerAndMorph() {
    try {
      const response = await fetch(window.location.pathname);
      if (response.ok) {
        const html = await response.text();
        await doMorph(html);
        isChecking = false;
        connect();
      } else {
        setTimeout(checkServerAndMorph, 200);
      }
    } catch (e) {
      setTimeout(checkServerAndMorph, 200);
    }
  }

  async function doMorph(html) {
    if (window.Idiomorph && window.htmx) {
      try {
        if (!html) {
          const response = await fetch(window.location.pathname);
          html = await response.text();
        }
        
        const parser = new DOMParser();
        const newDoc = parser.parseFromString(html, 'text/html');
        const newBody = newDoc.body;

        Idiomorph.morph(document.body, newBody, {
          morphStyle: 'innerHTML',
          callbacks: {
            beforeNodeMorphed: (oldNode, newNode) => {
              if (oldNode instanceof HTMLDialogElement && oldNode.open) {
                newNode.setAttribute('open', '');
              }
              if (newNode.tagName === 'SCRIPT' && newNode.innerText.includes('connect()')) {
                return false; 
              }
              return true;
            }
          }
        });

        htmx.process(document.body);
        
        console.log("🍲 Stew: Body morphed with Idiomorph directly");
      } catch (e) {
        console.error("Stew Morph Error:", e);
        window.location.reload();
      }
    } else {
      window.location.reload();
    }
  }

  document.addEventListener('DOMContentLoaded', () => {
    connect();
  });
</script>

<style>
  #stew-reload-trigger.htmx-request { display: none !important; }
</style>
`

func InjectScript() string {
	if os.Getenv("STEW_DEV") == "true" {
		return liveReloadScript
	}
	return ""
}

func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/stew/live" {
			SSEHandler(w, r)
			return
		}

		next.ServeHTTP(w, r)
	})
}
