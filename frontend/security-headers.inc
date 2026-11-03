# Incluído em cada location do nginx.conf.
# CSP: scripts só do próprio site (um XSS não carrega código de fora nem lê o token
# para outro endereço); fontes do Google Fonts; imagens de qualquer https (miniaturas
# das prévias de links); o site não pode ser aberto dentro de iframe de outro.
add_header Content-Security-Policy "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; font-src 'self' https://fonts.gstatic.com; img-src 'self' data: https:; connect-src 'self'; object-src 'none'; base-uri 'self'; form-action 'self'; frame-ancestors 'none'" always;
add_header X-Content-Type-Options "nosniff" always;
add_header X-Frame-Options "DENY" always;
add_header Referrer-Policy "strict-origin-when-cross-origin" always;
add_header Permissions-Policy "camera=(), microphone=(), geolocation=()" always;
