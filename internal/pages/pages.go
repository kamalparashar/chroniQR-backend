package pages

import "strings"

// NotFound returns the 404 branded HTML page.
func NotFound() string {
	return branded("🔍", "QR Code Not Found", "This QR code doesn't exist or the link may be incorrect.")
}

// Inactive returns the 410 branded HTML page.
func Inactive() string {
	return branded("⏸️", "This QR Code is No Longer Active", "The owner has deactivated or expired this QR code.")
}

// Error returns the 500 branded HTML page.
func Error() string {
	return branded("⚠️", "Something Went Wrong", "We couldn't process this QR code. Please try again later.")
}

func branded(icon, title, body string) string {
	return `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1.0" />
  <title>` + title + ` — chroniQR</title>
  <link rel="preconnect" href="https://fonts.googleapis.com">
  <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
  <link href="https://fonts.googleapis.com/css2?family=Outfit:wght@400;600;700&display=swap" rel="stylesheet">
  <style>
    * { margin: 0; padding: 0; box-sizing: border-box; }
    body { 
      font-family: 'Outfit', -apple-system, sans-serif; 
      background: radial-gradient(circle at center, #1e1b4b 0%, #09090b 100%); 
      color: #f4f4f5;
      display: flex; 
      align-items: center; 
      justify-content: center; 
      min-height: 100vh;
      overflow: hidden;
    }
    .card { 
      text-align: center; 
      padding: 56px 40px; 
      max-width: 440px; 
      border-radius: 24px;
      background: rgba(255, 255, 255, 0.03);
      backdrop-filter: blur(16px);
      -webkit-backdrop-filter: blur(16px);
      border: 1px solid rgba(255, 255, 255, 0.08);
      box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.5);
      animation: fadeIn 0.8s cubic-bezier(0.16, 1, 0.3, 1);
    }
    @keyframes fadeIn {
      from { opacity: 0; transform: translateY(20px); }
      to { opacity: 1; transform: translateY(0); }
    }
    .icon { 
      font-size: 72px; 
      margin-bottom: 24px; 
      display: inline-block;
      animation: float 3s ease-in-out infinite;
    }
    @keyframes float {
      0%, 100% { transform: translateY(0); }
      50% { transform: translateY(-10px); }
    }
    h1 { 
      font-size: 26px; 
      font-weight: 700; 
      margin-bottom: 16px; 
      background: linear-gradient(135deg, #a5b4fc 0%, #818cf8 100%);
      -webkit-background-clip: text;
      -webkit-text-fill-color: transparent;
    }
    p { color: #a1a1aa; font-size: 16px; line-height: 1.6; }
    .brand { 
      margin-top: 48px; 
      font-size: 14px; 
      color: #52525b; 
      letter-spacing: 0.5px;
    }
    .brand span { 
      background: linear-gradient(135deg, #818cf8 0%, #c084fc 100%);
      -webkit-background-clip: text;
      -webkit-text-fill-color: transparent;
      font-weight: 700; 
    }
  </style>
</head>
<body>
  <div class="card">
    <div class="icon">` + icon + `</div>
    <h1>` + title + `</h1>
    <p>` + body + `</p>
    <div class="brand">Powered by <span>chroniQR</span></div>
  </div>
</body>
</html>`
}

// EmailLanding returns a landing page that opens the user's mail client.
func EmailLanding(mailtoURL string) string {
	return strings.Join([]string{
		`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1.0" />
  <title>Opening Mail — chroniQR</title>
  <link rel="preconnect" href="https://fonts.googleapis.com">
  <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
  <link href="https://fonts.googleapis.com/css2?family=Outfit:wght@400;600;700&display=swap" rel="stylesheet">
  <style>
    * { margin: 0; padding: 0; box-sizing: border-box; }
    body { 
      font-family: 'Outfit', -apple-system, sans-serif; 
      background: radial-gradient(circle at center, #1e1b4b 0%, #09090b 100%); 
      color: #f4f4f5;
      display: flex; 
      align-items: center; 
      justify-content: center; 
      min-height: 100vh; 
    }
    .card { 
      text-align: center; 
      padding: 56px 40px; 
      max-width: 440px; 
      width: 100%; 
      border-radius: 24px;
      background: rgba(255, 255, 255, 0.03);
      backdrop-filter: blur(16px);
      -webkit-backdrop-filter: blur(16px);
      border: 1px solid rgba(255, 255, 255, 0.08);
      box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.5);
      animation: fadeIn 0.8s cubic-bezier(0.16, 1, 0.3, 1);
    }
    @keyframes fadeIn {
      from { opacity: 0; transform: translateY(20px); }
      to { opacity: 1; transform: translateY(0); }
    }
    .icon { 
      font-size: 64px; 
      margin-bottom: 24px; 
      display: inline-block;
      animation: pulse 2s infinite;
    }
    @keyframes pulse {
      0%, 100% { transform: scale(1); }
      50% { transform: scale(1.1); }
    }
    h1 { 
      font-size: 24px; 
      font-weight: 700; 
      margin-bottom: 16px; 
      background: linear-gradient(135deg, #a5b4fc 0%, #818cf8 100%);
      -webkit-background-clip: text;
      -webkit-text-fill-color: transparent;
    }
    p { color: #a1a1aa; font-size: 16px; line-height: 1.6; margin-bottom: 36px; }
    .btn { 
      display: inline-block; 
      background: linear-gradient(135deg, #6366f1 0%, #4f46e5 100%); 
      color: #fff; 
      font-size: 17px;
      font-weight: 600; 
      padding: 16px 40px; 
      border-radius: 14px;
      text-decoration: none; 
      width: 100%; 
      box-shadow: 0 10px 15px -3px rgba(99, 102, 241, 0.3);
      transition: all 0.3s ease;
    }
    .btn:hover {
      transform: translateY(-2px);
      box-shadow: 0 20px 25px -5px rgba(99, 102, 241, 0.4);
    }
    .brand { 
      margin-top: 48px; 
      font-size: 14px; 
      color: #52525b; 
    }
    .brand span { 
      background: linear-gradient(135deg, #818cf8 0%, #c084fc 100%);
      -webkit-background-clip: text;
      -webkit-text-fill-color: transparent;
      font-weight: 700; 
    }
  </style>
</head>
<body>
  <div class="card">
    <div class="icon">✉️</div>
    <h1>Opening your mail app...</h1>
    <p>Your mail app should open automatically. If it doesn't, tap the button below.</p>
    <a class="btn" href="`, mailtoURL, `">Open Mail App</a>
    <div class="brand">Powered by <span>chroniQR</span></div>
  </div>
  <script>
    window.addEventListener('load', function () {
      setTimeout(function () { window.location.href = '`, mailtoURL, `'; }, 500);
    });
  </script>
</body>
</html>`,
	}, "")
}

// CallLanding returns an AI voice call landing page.
func CallLanding(callerNumber, ctaText string) string {
	if ctaText == "" {
		ctaText = "Tap to connect with our AI assistant"
	}
	return strings.Join([]string{
		`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1.0" />
  <title>Connect — chroniQR</title>
  <link rel="preconnect" href="https://fonts.googleapis.com">
  <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
  <link href="https://fonts.googleapis.com/css2?family=Outfit:wght@400;600;700&display=swap" rel="stylesheet">
  <style>
    * { margin: 0; padding: 0; box-sizing: border-box; }
    body { 
      font-family: 'Outfit', -apple-system, sans-serif; 
      background: radial-gradient(circle at center, #1e1b4b 0%, #09090b 100%); 
      color: #f4f4f5;
      display: flex; 
      align-items: center; 
      justify-content: center; 
      min-height: 100vh; 
    }
    .card { 
      text-align: center; 
      padding: 56px 40px; 
      max-width: 440px; 
      width: 100%; 
      border-radius: 24px;
      background: rgba(255, 255, 255, 0.03);
      backdrop-filter: blur(16px);
      -webkit-backdrop-filter: blur(16px);
      border: 1px solid rgba(255, 255, 255, 0.08);
      box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.5);
      animation: fadeIn 0.8s cubic-bezier(0.16, 1, 0.3, 1);
    }
    @keyframes fadeIn {
      from { opacity: 0; transform: translateY(20px); }
      to { opacity: 1; transform: translateY(0); }
    }
    .icon { 
      font-size: 64px; 
      margin-bottom: 24px; 
      display: inline-block;
      animation: wiggle 2s infinite;
    }
    @keyframes wiggle {
      0%, 100% { transform: rotate(0); }
      15% { transform: rotate(-15deg); }
      30% { transform: rotate(10deg); }
      45% { transform: rotate(-5deg); }
      60% { transform: rotate(5deg); }
      75% { transform: rotate(0); }
    }
    h1 { 
      font-size: 24px; 
      font-weight: 700; 
      margin-bottom: 16px; 
      background: linear-gradient(135deg, #a5b4fc 0%, #818cf8 100%);
      -webkit-background-clip: text;
      -webkit-text-fill-color: transparent;
    }
    p { color: #a1a1aa; font-size: 16px; line-height: 1.6; margin-bottom: 36px; }
    .btn { 
      display: inline-block; 
      background: linear-gradient(135deg, #6366f1 0%, #4f46e5 100%); 
      color: #fff; 
      font-size: 17px;
      font-weight: 600; 
      padding: 16px 40px; 
      border-radius: 14px;
      text-decoration: none; 
      width: 100%; 
      box-shadow: 0 10px 15px -3px rgba(99, 102, 241, 0.3);
      transition: all 0.3s ease;
    }
    .btn:hover {
      transform: translateY(-2px);
      box-shadow: 0 20px 25px -5px rgba(99, 102, 241, 0.4);
    }
    .desktop-form { display: none; margin-top: 24px; }
    .desktop-form input { 
      width: 100%; 
      padding: 14px 18px; 
      border-radius: 12px;
      border: 1px solid rgba(255, 255, 255, 0.1); 
      background: rgba(255, 255, 255, 0.05); 
      color: #f4f4f5;
      font-size: 16px; 
      margin-bottom: 16px; 
      transition: border 0.3s;
    }
    .desktop-form input:focus {
      outline: none;
      border-color: #6366f1;
    }
    .desktop-form button { 
      width: 100%; 
      padding: 14px; 
      background: linear-gradient(135deg, #6366f1 0%, #4f46e5 100%); 
      color: #fff;
      border: none; 
      border-radius: 12px; 
      font-size: 16px; 
      font-weight: 600;
      cursor: pointer; 
      box-shadow: 0 10px 15px -3px rgba(99, 102, 241, 0.3);
      transition: all 0.3s ease;
    }
    .desktop-form button:hover {
      transform: translateY(-2px);
      box-shadow: 0 20px 25px -5px rgba(99, 102, 241, 0.4);
    }
    .brand { 
      margin-top: 48px; 
      font-size: 14px; 
      color: #52525b; 
    }
    .brand span { 
      background: linear-gradient(135deg, #818cf8 0%, #c084fc 100%);
      -webkit-background-clip: text;
      -webkit-text-fill-color: transparent;
      font-weight: 700; 
    }
    @media (hover: hover) and (pointer: fine) {
      .mobile-cta { display: none; }
      .desktop-form { display: block; }
    }
  </style>
</head>
<body>
  <div class="card">
    <div class="icon">📞</div>
    <h1>`, ctaText, `</h1>
    <p>Our AI assistant is ready to help you right now.</p>
    <div class="mobile-cta">
      <a class="btn" href="tel:`, callerNumber, `">Tap to Connect</a>
    </div>
    <div class="desktop-form">
      <p style="margin-bottom:16px; color:#a1a1aa">Enter your number and we'll call you:</p>
      <input type="tel" id="phone-input" placeholder="+1 (555) 000-0000" />
      <button onclick="requestCall()">Call Me</button>
    </div>
    <div class="brand">Powered by <span>chroniQR</span></div>
  </div>
  <script>
    function requestCall() {
      var phone = document.getElementById('phone-input').value.trim();
      if (!phone) return alert('Please enter your phone number');
      alert('We will call you at ' + phone + ' shortly!');
    }
  </script>
</body>
</html>`,
	}, "")
}

// GeolocationPage returns a branded interstitial page that:
// 1. Collects all client-side device info via built-in browser APIs
// 2. Uses navigator.geolocation to get precise GPS
// 3. POSTs everything to /api/scan-location
// 4. Redirects to the final destination URL
func GeolocationPage(scanID, finalURL string) string {
	return `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8"/>
  <meta name="viewport" content="width=device-width, initial-scale=1.0"/>
  <title>Redirecting — chroniQR</title>
  <link rel="preconnect" href="https://fonts.googleapis.com">
  <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
  <link href="https://fonts.googleapis.com/css2?family=Outfit:wght@400;600;700&display=swap" rel="stylesheet">
  <style>
    * { margin: 0; padding: 0; box-sizing: border-box; }
    body { 
      font-family: 'Outfit', -apple-system, sans-serif; 
      background: radial-gradient(circle at center, #1e1b4b 0%, #09090b 100%); 
      color: #f4f4f5;
      display: flex; 
      align-items: center; 
      justify-content: center; 
      min-height: 100vh; 
    }
    .card { 
      text-align: center; 
      padding: 56px 40px; 
      max-width: 440px; 
      width: 100%; 
      border-radius: 24px;
      background: rgba(255, 255, 255, 0.03);
      backdrop-filter: blur(16px);
      -webkit-backdrop-filter: blur(16px);
      border: 1px solid rgba(255, 255, 255, 0.08);
      box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.5);
    }
    .spinner { 
      width: 60px; 
      height: 60px; 
      border: 4px solid rgba(255, 255, 255, 0.05); 
      border-top: 4px solid #6366f1;
      border-radius: 50%; 
      animation: spin 0.8s cubic-bezier(0.5, 0.1, 0.4, 0.9) infinite; 
      margin: 0 auto 24px; 
    }
    @keyframes spin { to { transform: rotate(360deg); } }
    h1 { 
      font-size: 22px; 
      font-weight: 600; 
      margin-bottom: 8px; 
      background: linear-gradient(135deg, #a5b4fc 0%, #818cf8 100%);
      -webkit-background-clip: text;
      -webkit-text-fill-color: transparent;
    }
    p { color: #a1a1aa; font-size: 15px; line-height: 1.5; }
    .brand { 
      margin-top: 48px; 
      font-size: 14px; 
      color: #52525b; 
    }
    .brand span { 
      background: linear-gradient(135deg, #818cf8 0%, #c084fc 100%);
      -webkit-background-clip: text;
      -webkit-text-fill-color: transparent;
      font-weight: 700; 
    }
  </style>
</head>
<body>
  <div class="card">
    <div class="spinner"></div>
    <h1>Redirecting you...</h1>
    <p>Please wait a moment</p>
    <div class="brand">Powered by <span>chroniQR</span></div>
  </div>
  <script>
    (function(){
      var scanId = "` + scanID + `";
      var finalUrl = "` + finalURL + `";
      var done = false;

      function getDeviceMeta(){
        var meta = {};
        try{ meta.screen_width = screen.width; }catch(e){}
        try{ meta.screen_height = screen.height; }catch(e){}
        try{ meta.timezone = Intl.DateTimeFormat().resolvedOptions().timeZone; }catch(e){}
        try{ meta.language = navigator.language || navigator.userLanguage || ''; }catch(e){}
        try{ meta.platform = navigator.platform || ''; }catch(e){}
        try{ meta.color_depth = screen.colorDepth || 0; }catch(e){}
        try{ meta.pixel_ratio = window.devicePixelRatio || 1; }catch(e){}
        try{ meta.touch_support = ('ontouchstart' in window) || (navigator.maxTouchPoints > 0); }catch(e){}
        try{
          var conn = navigator.connection || navigator.mozConnection || navigator.webkitConnection;
          meta.connection_type = conn ? (conn.effectiveType || conn.type || '') : '';
        }catch(e){}
        try{ meta.online = navigator.onLine; }catch(e){}
        try{ meta.cookies_enabled = navigator.cookieEnabled; }catch(e){}
        return meta;
      }

      function sendData(payload){
        var body = JSON.stringify(payload);
        if(navigator.sendBeacon){
          navigator.sendBeacon('/api/scan-location', new Blob([body],{type:'application/json'}));
        } else {
          var xhr = new XMLHttpRequest();
          xhr.open('POST','/api/scan-location',true);
          xhr.setRequestHeader('Content-Type','application/json');
          xhr.send(body);
        }
      }

      function goToDestination(){
        if(!done){ done=true; window.location.href=finalUrl; }
      }

      setTimeout(goToDestination, 3000);

      var meta = getDeviceMeta();

      if(navigator.geolocation){
        navigator.geolocation.getCurrentPosition(
          function(pos){
            var payload = {scan_id:scanId, lat:pos.coords.latitude, lng:pos.coords.longitude};
            for(var k in meta){ payload[k]=meta[k]; }
            sendData(payload);
            goToDestination();
          },
          function(err){
            var payload = {scan_id:scanId, lat:0, lng:0};
            for(var k in meta){ payload[k]=meta[k]; }
            sendData(payload);
            goToDestination();
          },
          {enableHighAccuracy:true, timeout:2500, maximumAge:0}
        );
      } else {
        var payload = {scan_id:scanId, lat:0, lng:0};
        for(var k in meta){ payload[k]=meta[k]; }
        sendData(payload);
        goToDestination();
      }
    })();
  </script>
</body>
</html>`
}

// GeoBackgroundScript returns a script tag that silently attempts to capture
// GPS + full device meta in the background for landing pages (email, call).
func GeoBackgroundScript(scanID string) string {
	if scanID == "" {
		return ""
	}
	return `<script>
(function(){
  function getDeviceMeta(){
    var m={};
    try{m.screen_width=screen.width}catch(e){}
    try{m.screen_height=screen.height}catch(e){}
    try{m.timezone=Intl.DateTimeFormat().resolvedOptions().timeZone}catch(e){}
    try{m.language=navigator.language||''}catch(e){}
    try{m.platform=navigator.platform||''}catch(e){}
    try{m.color_depth=screen.colorDepth||0}catch(e){}
    try{m.pixel_ratio=window.devicePixelRatio||1}catch(e){}
    try{m.touch_support=('ontouchstart' in window)||(navigator.maxTouchPoints>0)}catch(e){}
    try{var c=navigator.connection||navigator.mozConnection||navigator.webkitConnection;m.connection_type=c?(c.effectiveType||c.type||''):''}catch(e){}
    try{m.online=navigator.onLine}catch(e){}
    try{m.cookies_enabled=navigator.cookieEnabled}catch(e){}
    return m;
  }
  function send(p){
    var b=JSON.stringify(p);
    if(navigator.sendBeacon){navigator.sendBeacon('/api/scan-location',new Blob([b],{type:'application/json'}))}
    else{var x=new XMLHttpRequest();x.open('POST','/api/scan-location',true);x.setRequestHeader('Content-Type','application/json');x.send(b)}
  }
  var meta=getDeviceMeta();
  if(navigator.geolocation){
    navigator.geolocation.getCurrentPosition(function(pos){
      var p={scan_id:"` + scanID + `",lat:pos.coords.latitude,lng:pos.coords.longitude};
      for(var k in meta){p[k]=meta[k]}
      send(p);
    },function(){
      var p={scan_id:"` + scanID + `",lat:0,lng:0};
      for(var k in meta){p[k]=meta[k]}
      send(p);
    },{enableHighAccuracy:true,timeout:5000,maximumAge:0});
  } else {
    var p={scan_id:"` + scanID + `",lat:0,lng:0};
    for(var k in meta){p[k]=meta[k]}
    send(p);
  }
})();
</script>`
}
