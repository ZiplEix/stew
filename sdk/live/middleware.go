package live

import (
	"net/http"
	"os"
)

const liveReloadScript = `
<script>
  function connect() {
    const ev = new EventSource('/stew/live');
    
    ev.onmessage = (event) => {
      if (event.data === 'reload') {
        window.location.reload();
      }
    };

    ev.onerror = () => {
      ev.close();
      setTimeout(() => {
        window.location.reload();
      }, 700);
    };
  }
  connect();
</script>
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
