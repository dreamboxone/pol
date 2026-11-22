'use strict';
'require view';
'require rpc';
'require poll';
'require dom';

function node(tag,attrs,children){function clean(v){return Array.isArray(v)?v.filter(function(x){return x!==null&&x!==undefined;}).map(clean):v;}return E(tag,attrs,children==null?[]:clean(children));}

var api = {};
['status', 'settings', 'logs', 'save', 'toggle', 'import', 'register', 'deploy', 'releases', 'update', 'scan', 'geo', 'geoUpdate', 'check', 'scanControl', 'torBridges', 'cfDelete', 'cfUpdate', 'cfSignIn', 'cfAutoUpdate', 'cfUsage', 'cfDomain', 'config', 'dnsProbe'].forEach(function(name) {
	var params = {cfDomain:['token','account','name','kind','hostname','action'],cfUsage:['token','account','name'],cfAutoUpdate:['token','account','enabled','kind'],cfSignIn:['token','account','kind'],cfUpdate:['token','account','name','kind'],dnsProbe:['request'],cfDelete:['token','account','name','kind'],torBridges:['transport'],scanControl:['action'],releases:['component'],check:['site'], geoUpdate:['kind'], scan:['text','options'], logs:['kind'], save:['settings'], toggle:['enabled'], import:['text'], register:['kind'], deploy:['token','account','name','kind'], update:['component','tag']}[name] || [];
	api[name] = rpc.declare({object:'pol', method:name, params:params, reject:true});
});

function transientRPC(error){return /Object not found|error -32000/.test(String(error&&error.message||error));}
var rawStatus=api.status;
api.status=function(){var attempt=0;function read(){return rawStatus().catch(function(error){if(!transientRPC(error)||++attempt>=4)throw error;return new Promise(function(resolve){setTimeout(resolve,1000);}).then(read);});}return read();};

var iconPaths = {"home":"M3 10 12 3l9 7v11h-6v-7H9v7H3Z","connection":"M7 7h10v10H7ZM9 3v4m6-4v4M9 17v4m6-4v4M3 9h4m-4 6h4m10-6h4m-4 6h4","cloudflare":"M6 18h12a4 4 0 0 0 0-8 6 6 0 0 0-12-1 4.5 4.5 0 0 0 0 9Z","routing":"M4 4h6v6H4Zm10 10h6v6h-6ZM7 10v7h7M10 7h7v7","reports":"M5 3h14v18H5ZM9 7h6M9 11h6m-6 4h4","updates":"M12 3v12m-5-5 5 5 5-5M4 17v4h16v-4","about":"M4 4h7a3 3 0 0 1 3 3v14a3 3 0 0 0-3-3H4Zm10 3a3 3 0 0 1 3-3h4v14h-4a3 3 0 0 0-3 3","power":"M12 3v9M6.3 6.3a8 8 0 1 0 11.4 0","light":"M12 2v2m0 16v2M2 12h2m16 0h2M5 5l1 1m12 12 1 1M5 19l1-1M18 6l1-1M16 12a4 4 0 1 1-8 0 4 4 0 0 1 8 0","dark":"M21 13A9 9 0 1 1 11 3a7 7 0 0 0 10 10Z","auto":"M3 4h18v14H3Zm5 18h8m-4-4v4","telegram":"M21 3 3 10l7 3 3 8Zm-11 10L21 3","source":"m8 6-6 6 6 6m8-12 6 6-6 6","refresh":"M20 10a8 8 0 0 0-14-3M4 3v5h5M4 14a8 8 0 0 0 14 3m2 4v-5h-5","import":"M12 3v12m-5-5 5 5 5-5M4 17v4h16v-4","export":"M12 15V3m-5 5 5-5 5 5M4 17v4h16v-4","check":"m5 12 5 5 9-10","paused":"M8 4v16M16 4v16","live":"m8 4 12 8-12 8Z"};
iconPaths.trash='M3 6h18M9 6V3h6v3M5 6l1 15h12l1-15M10 10v7m4-7v7';
iconPaths.restore=iconPaths.import;
iconPaths.eye='M2 12s3.5-7 10-7 10 7 10 7-3.5 7-10 7S2 12 2 12Z M15 12a3 3 0 1 1-6 0 3 3 0 0 1 6 0';
function icon(name) {
	var span=node('span',{'class':'pol-icon','aria-hidden':'true'});
	var svg=document.createElementNS('http://www.w3.org/2000/svg','svg'),p=document.createElementNS(svg.namespaceURI,'path');
	svg.setAttribute('viewBox','0 0 24 24');svg.setAttribute('fill','none');svg.setAttribute('stroke','currentColor');svg.setAttribute('stroke-width','1.8');svg.setAttribute('stroke-linecap','round');svg.setAttribute('stroke-linejoin','round');p.setAttribute('d',iconPaths[name]||iconPaths.check);svg.appendChild(p);span.appendChild(svg);return span;
}

iconPaths.settings='M9.5 3h5l.6 2.5 1.5.9 2.5-.7 2.5 4.3-1.9 1.8v1.7l1.9 1.8-2.5 4.3-2.5-.7-1.5.9-.6 2.2h-5l-.6-2.2-1.5-.9-2.5.7L2.4 15l1.9-1.8v-1.7L2.4 9.7l2.5-4.3 2.5.7 1.5-.9Z M15.5 12a3.5 3.5 0 1 1-7 0 3.5 3.5 0 0 1 7 0';
iconPaths.globe='M3 12a9 9 0 1 0 18 0 9 9 0 1 0-18 0M3 12h18M12 3a17 17 0 0 1 0 18M12 3a17 17 0 0 0 0 18';
iconPaths.pulse='M3 12h4l3-8 4 16 3-8h4';
iconPaths.video='M3 5h18v14H3Zm7 4 6 3-6 3Z';
var words = {
	fa: {
		connectionCheck:'بررسی اتصال', checkNow:'بررسی اتصال', state:'وضعیت', coreRunning:'در حال اجرا', coreStopped:'اجرا نمی‌شود', nodeName:'نام اتصال', help:'راهنمای این گزینه', routeIran:'تفکیک ترافیک ایران', iranDns:'DNS مستقیم ایران', geoTitle:'فایل‌های مسیریابی', geoSource:'منبع دریافت', customSource:'نشانی سفارشی', geoUpdate:'به‌روزرسانی فایل', installedSource:'منبع فایل نصب‌شده', updatedAt:'آخرین به‌روزرسانی', fileSize:'حجم فایل', notDownloaded:'دریافت نشده', usage:'میزان مصرف (GB)', today:'امروز', week:'۷ روز', month:'۳۰ روز', upload:'ارسال', downloadBytes:'دریافت', total:'مجموع', connectionStatus:'وضعیت اتصال', internetOk:'متصل', internetWaiting:'در انتظار بررسی اتصال', internetFailed:'بررسی اتصال ناموفق بود', server:'سرور', ping:'پینگ', exitIp:'IP خروجی', checkedAt:'زمان بررسی', uptime:'مدت فعالیت روتر', days:'روز', scan:'اسکن اتصال', scanHelp:'حداکثر ۱۲ IP یا دامنه، هرکدام در یک خط. برای WARP و MASQUE از host:port استفاده کنید. هر مورد با درخواست HTTPS از داخل هسته آزمایش می‌شود؛ اسکن خودکار ۲۴ آدرس را از بازه‌های BPB انتخاب و آزمایش می‌کند. اسکن ممکن است چند دقیقه طول بکشد و تنظیم فعال را تغییر نمی‌دهد.', useEndpoint:'استفاده', scanEmpty:'هنوز اسکن انجام نشده است.', failed:'ناموفق', auto:'همراه تم دستگاه', support:'پشتیبانی تلگرام', source:'کد منبع', guide:'راهنمای کامل', fingerprint:'فینگرپرینت TLS', workerSni:'SNI ورکر', echDns:'DNS برای ECH', echHelp:'اختیاری؛ IP یا نشانی HTTPS سرویس DoH. با فرگمنت هم‌زمان فعال نمی‌شود.', tcpFastOpen:'TCP Fast Open', dnsStrategy:'اولویت آدرس DNS', noiseSize:'اندازه نویز UDP', noiseDelay:'تاخیر نویز (میلی‌ثانیه)', noiseReset:'بازه بازنشانی نویز', warpMtu:'MTU وارپ', warpKeepAlive:'Keepalive (ثانیه)', warpReserved:'استفاده از بایت‌های Reserved اکانت', advanced:'تنظیمات پیشرفته', advancedHelp:'مقادیر روی اتصال‌های ساخته‌شده در پل اعمال می‌شوند. کانفیگ واردشده تنظیمات خروجی خودش را حفظ می‌کند.', UseIPv4:'فقط IPv4', UseIPv6:'فقط IPv6', UseIP:'IPv4 و IPv6', title:'پل', subtitle:'اتصال امن، کنترل ساده', home:'خانه', connection:'تنظیمات پایه', cloudflare:'کلادفلر', routing:'مدیریت ترافیک', reports:'گزارش', updates:'به‌روزرسانی', about:'درباره',
		main:'دکمه اصلی', on:'روشن', off:'خاموش', connected:'هسته فعال است', disconnected:'هسته خاموش است', starting:'در حال اجرا…', applied:'تغییرات اعمال شد', applying:'در حال اعمال…', error:'خطا',
		intro:'یک اتصال، برای همه دستگاه‌های شبکه', introText:'نوع اتصال را انتخاب کنید و با دکمه اصلی، سرویس را روشن کنید. تغییرات به‌صورت مستقیم اعمال می‌شوند.',
		mode:'نوع اتصال', variant:'روش اتصال', normal:'معمولی', fragment:'فرگمنت', obfuscated:'مبهم‌سازی', worker:'ورکر', warp:'وارپ', masque:'مسک', serverless:'بدون سرور', tor:'تور', psiphon:'سایفون', custom:'کانفیگ واردشده',
		host:'دامنه Worker', uuid:'شناسه UUID', address:'آدرس اتصال / IP تمیز', port:'پورت', path:'مسیر WebSocket', dns:'DNS راه‌دور', lan:'اینترفیس شبکه محلی', transparent:'اتصال تمام دستگاه‌های شبکه', length:'طول فرگمنت', delay:'تاخیر فرگمنت (میلی‌ثانیه)', endpoint:'نقطه اتصال', transport:'انتقال MASQUE', sni:'نام سرور TLS', chain:'پروکسی زنجیره‌ای (URI)', logLevel:'سطح گزارش هسته',
		importTitle:'وارد کردن کانفیگ', importHelp:'لینک VLESS، Trojan، SOCKS یا کانفیگ JSON هسته Xray و تنظیمات خروجی همین برنامه را وارد کنید.', import:'وارد کردن', paste:'کانفیگ را اینجا قرار دهید…', optional:'اختیاری',
		account:'اکانت اتصال', registered:'آماده', missing:'ثبت نشده', registerWarp:'ساخت اکانت WARP', registerMasque:'ساخت اکانت MASQUE', registrationHelp:'ثبت نام اکانت با سرویس Cloudflare انجام می‌شود. دسترسی اینترنت روتر لازم است.',
		cfTitle:'ساخت ورکر اختصاصی', cfHelp:'توکن API با مجوز Workers Scripts Edit یا نقش Workers Admin و شناسه حساب Cloudflare را وارد کنید. توکن ذخیره نمی‌شود. ابتدا زیر‌دامنه workers.dev را در پنل Cloudflare فعال کنید.', token:'توکن API', accountId:'اکانت آی‌دی (Account ID)', workerName:'نام جدید Worker', deploy:'ساخت Worker', deployed:'Worker ساخته شد', tokenLink:'ساخت توکن در Cloudflare',
		bypass:'دامنه‌های مستقیم', block:'دامنه‌های مسدود', domainHelp:'هر دامنه در یک خط؛ مانند example.com. نشانی‌های شبکه خصوصی مستقیم عبور می‌کنند.', bypassHelp:'ترافیک این دامنه‌ها بدون تونل عبور می‌کند.',
		all:'همه', app:'برنامه', core:'هسته', refresh:'تازه‌سازی', download:'دریافت گزارش', emptyLogs:'هنوز رخدادی ثبت نشده است.', paused:'توقف نمایش زنده', live:'نمایش زنده',
		updateTitle:'به‌روزرسانی', updateHelp:'به‌روزرسانی هسته، رابط را تغییر نمی‌دهد. بسته برنامه مطابق مدیر بسته روتر انتخاب می‌شود و هسته را جایگزین نمی‌کند.', current:'نسخه نصب‌شده', latest:'آخرین نسخه', check:'بررسی', updateCore:'به‌روزرسانی هسته', updateApp:'به‌روزرسانی برنامه', repository:'مخزن انتشار برنامه (owner/name)', repoHelp:'مخزن رسمی dreamboxone/pol است. فقط برای انتشار شخصی آن را تغییر دهید.', noRepo:'مخزن انتشار هنوز تنظیم نشده است.', verifying:'در حال دریافت و بررسی…', scheduled:'نصب برنامه شروع شد؛ پس از چند لحظه صفحه را تازه‌سازی کنید و گزارش را ببینید.', done:'به‌روزرسانی انجام شد',
		proxy:'پروکسی محلی', firewall:'مسیریابی شبکه', active:'فعال', inactive:'غیرفعال', package:'بسته‌بندی', noApply:'اعمال مستقیم تنظیمات', aboutText:'نسخه مخصوص OpenWrt بر پایه BPB-VPN. رابط مدیریتی LuCI با هسته مستقل Xray، پشتیبانی از IPK و APK و زبان فارسی و انگلیسی.',
		license:'مجوز و منابع', localOnly:'دسترسی پروکسی محلی: فقط خود روتر', notice:'حالت بدون سرور با فرگمنت کار می‌کند و IP خروجی را تغییر نمی‌دهد. موتورهای Tor و سایفون از داخل پل مدیریت می‌شوند. برقراری اتصال به وضعیت شبکه و پل‌های Tor بستگی دارد.', configNotice:'در وارد کردن JSON، خروجی‌ها، DNS و قوانین مسیریابی Xray حفظ می‌شوند؛ ورودی‌های دستگاه با ورودی‌های مخصوص روتر جایگزین می‌شوند.', manual:'راهنمای اتصال', manualText:'۱. Worker بسازید یا کانفیگ وارد کنید. ۲. نوع اتصال را انتخاب کنید. ۳. دکمه اصلی را روشن کنید.',
		debug:'اشکال‌زدایی', info:'اطلاعات', warning:'هشدار', none:'بدون گزارش', light:'روشن', dark:'تاریک', export:'پشتیبان‌گیری', backupTitle:'پشتیبان‌گیری و بازیابی', savedWorker:'ورکر ذخیره‌شده', deleteWorker:'حذف از پل', restore:'بازیابی', exportHelp:'فایل خروجی شامل کلیدها و شناسه‌های اتصال است؛ آن را محرمانه نگه دارید.',
	},
	en: {
		connectionCheck:'Connection check', checkNow:'Check', state:'Status', coreRunning:'RUNNING', coreStopped:'NOT RUNNING', nodeName:'Connection', help:'Option help', routeIran:'Bypass Iranian traffic', iranDns:'Iran direct DNS', geoTitle:'Routing data', geoSource:'Download source', customSource:'Custom URL', geoUpdate:'Update file', installedSource:'Installed file source', updatedAt:'Last updated', fileSize:'File size', notDownloaded:'Not downloaded', usage:'Traffic usage (GB)', today:'Today', week:'7 days', month:'30 days', upload:'Upload', downloadBytes:'Download', total:'Total', connectionStatus:'Connection status', internetOk:'Internet reachable', internetWaiting:'Waiting for connection check', internetFailed:'Connection check failed', server:'Server', ping:'Latency', exitIp:'Exit IP', checkedAt:'Last checked', uptime:'Router uptime', days:'days', scan:'Scan connection', scanHelp:'Up to 12 IPs or hostnames, one per line. Use host:port for WARP and MASQUE. Probes HTTPS through the actual core. May take a few minutes; leaves active settings in place.', useEndpoint:'USE', scanEmpty:'No scan results yet.', failed:'Failed', auto:'System theme', support:'Telegram support', source:'Source code', guide:'Complete guide', fingerprint:'TLS fingerprint', workerSni:'Worker SNI', echDns:'ECH DNS', echHelp:'Optional IP or HTTPS DoH URL. Disable before enabling fragmentation.', tcpFastOpen:'TCP Fast Open', dnsStrategy:'DNS address preference', noiseSize:'UDP noise size', noiseDelay:'Noise delay (milliseconds)', noiseReset:'Noise reset range', warpMtu:'WARP MTU', warpKeepAlive:'Keepalive (seconds)', warpReserved:'Use account reserved bytes', advanced:'BPB advanced settings', advancedHelp:'Applies to connections built by Pol. Imported configurations retain their own outbound settings.', UseIPv4:'IPv4 only', UseIPv6:'IPv6 only', UseIP:'IPv4 and IPv6', title:'Pol', subtitle:'Secure connection. Simple control.', home:'Home', connection:'Basic settings', cloudflare:'Cloudflare', routing:'Traffic management', reports:'Logs', updates:'Updates', about:'About',
		main:'Main Switch', on:'On', off:'Off', connected:'Core is running', disconnected:'Core is stopped', starting:'Working…', applied:'Changes applied', applying:'Applying…', error:'Error',
		intro:'One connection, for your whole network', introText:'Choose a connection and use the main button to start the service. Settings take effect immediately.',
		mode:'Connection type', variant:'Connection variant', normal:'Normal', fragment:'Fragment', obfuscated:'Obfuscated', worker:'Worker', warp:'WARP', masque:'MASQUE', serverless:'Serverless', tor:'Tor', psiphon:'Psiphon', custom:'Imported configuration',
		host:'Worker hostname', uuid:'UUID', address:'Connection address / clean IP', port:'Port', path:'WebSocket path', dns:'Remote DNS', lan:'LAN interface', transparent:'Connect every LAN device', length:'Fragment length', delay:'Fragment delay (milliseconds)', endpoint:'Endpoint', transport:'MASQUE transport', sni:'TLS server name', chain:'Chain proxy (URI)', logLevel:'Core log level',
		importTitle:'Import configuration', importHelp:'Paste a VLESS, Trojan or SOCKS link, Xray JSON, or settings exported by this app.', import:'Import', paste:'Paste configuration here…', optional:'Optional',
		account:'Tunnel account', registered:'Ready', missing:'Not registered', registerWarp:'Register WARP account', registerMasque:'Register MASQUE account', registrationHelp:'Registers with Cloudflare. The router needs internet access.',
		cfTitle:'Deploy your own Worker', cfHelp:'Enter a Cloudflare API token with Workers Scripts Edit or Workers Admin permissions and your Account ID. The token is never saved. Set up a workers.dev subdomain in Cloudflare first.', token:'API token', accountId:'Account ID', workerName:'New Worker name', deploy:'Deploy Worker', deployed:'Worker deployed', tokenLink:'Create a Cloudflare token',
		bypass:'Direct domains', block:'Blocked domains', domainHelp:'One domain per line, such as example.com. Private networks bypass the tunnel.', bypassHelp:'These domains connect directly, outside the tunnel.',
		all:'All', app:'Application', core:'Core', refresh:'Refresh', download:'Download logs', emptyLogs:'No events yet.', paused:'Pause live view', live:'Live view',
		updateTitle:'Update', updateHelp:'Core updates leave the interface in place. App updates select the router’s package format and keep the core in place.', current:'Installed', latest:'Latest', check:'Check', updateCore:'Update core', updateApp:'Update application', repository:'Application release repository (owner/name)', repoHelp:'The official repository is dreamboxone/pol. Change only for your own releases.', noRepo:'No release repository configured yet.', verifying:'Downloading and verifying…', scheduled:'App installation started. Refresh shortly and check Logs.', done:'Update completed',
		proxy:'Local proxy', firewall:'LAN routing', active:'Active', inactive:'Inactive', package:'Package format', noApply:'Settings apply immediately', aboutText:'An OpenWrt adaptation of BPB-VPN. A LuCI interface with an independent Xray core, IPK and APK packaging, and Persian and English languages.',
		license:'License and sources', localOnly:'Local proxy access: router only', notice:'Serverless fragmentation does not change your public IP. Tor and Psiphon engines are managed by Pol. Connectivity depends on the network and Tor bridges.', configNotice:'JSON import preserves Xray outbounds, DNS and routing. Device listeners are replaced with router listeners.', manual:'Getting connected', manualText:'1. Deploy a Worker or import a configuration. 2. Choose the connection type. 3. Turn on the main button.',
		debug:'Debug', info:'Info', warning:'Warning', none:'No logs', light:'Light', dark:'Dark', export:'Back up', backupTitle:'Settings backup and restore', savedWorker:'Saved Worker', deleteWorker:'Remove from Pol', restore:'Restore', exportHelp:'The export contains connection keys and identifiers. Keep it private.',
	}
};

Object.assign(words.fa,{customCdn:'CDN سفارشی',upstreamRelay:'رله TLS (host:port)',dnsServers:'سرورهای DNS',dnsHosts:'IPهای راه‌انداز DNS',fakeSniDns:'DNS با SNI جایگزین',fakeDns:'FakeDNS',serverlessMode:'روش بدون سرور',packet:'فرگمنت و نویز',dns:'DNS راه‌دور','domain-fronting':'Domain Fronting',maskRules:'قوانین پیشرفته فرگمنت و نویز',fragments:'فرگمنت Worker',udpNoises:'نویز WARP',masqueFragments:'فرگمنت MASQUE',masqueNoises:'نویز MASQUE',serverlessFragments:'فرگمنت بدون سرور',serverlessNoises:'نویز بدون سرور',bestPing:'انتخاب خودکار بهترین آدرس',cleanIps:'آدرس‌های کاندید Worker',endpoints:'endpointهای کاندید',probeInterval:'بازه بررسی بهترین آدرس (ثانیه)',deploymentType:'نوع استقرار',workerProxyIps:'Proxy IPهای ورکر',workerFallback:'نشانی صفحه جایگزین',scanPause:'توقف موقت',scanResume:'ادامه اسکن',scanStop:'پایان اسکن',exitMode:'خروجی نهایی',torBridgeMode:'روش پل Tor',exitRegion:'کشور خروجی',torBridges:'پل‌های Tor',scanAuto:'اسکن خودکار IP',updateAction:'به‌روز رسانی',settings:'تنظیمات',geoDirectory:'محل فایل‌های قوانین Geo',geoAuto:'به‌روزرسانی خودکار',geoipUpdate:'GeoIP',geositeUpdate:'GeoSite',disabled:'غیرفعال',daily:'روزانه',weekly:'هفتگی',directTitle:'آدرس‌ها و دامنه‌های مستقیم',directIps:'لیست IPهای مستقیم',rebindTitle:'سایت‌های دارای ضعف امنیتی Rebind Attack',rebindDomains:'دامنه‌ها',remove:'حذف',add:'افزودن',backendRequired:'برای استفاده از این بخش، بسته جدید پل را نصب کنید.'});
Object.assign(words.en,{customCdn:'Custom CDN',upstreamRelay:'TLS relay (host:port)',dnsServers:'DNS servers',dnsHosts:'DNS bootstrap IPs',fakeSniDns:'Alternate-SNI DNS',fakeDns:'FakeDNS',serverlessMode:'Serverless method',packet:'Fragmentation and noise','domain-fronting':'Domain fronting',maskRules:'Advanced fragmentation and noise rules',fragments:'Worker fragmentation',udpNoises:'WARP noise',masqueFragments:'MASQUE fragmentation',masqueNoises:'MASQUE noise',serverlessFragments:'Serverless fragmentation',serverlessNoises:'Serverless noise',bestPing:'Automatically select best endpoint',cleanIps:'Worker candidates',endpoints:'Endpoint candidates',probeInterval:'Probe interval (seconds)',deploymentType:'Deployment type',workerProxyIps:'Worker proxy IPs',workerFallback:'Fallback page URL',scanPause:'Pause scan',scanResume:'Resume scan',scanStop:'Stop scan',exitMode:'Final exit',torBridgeMode:'Tor bridge mode',exitRegion:'Exit country',torBridges:'Tor bridges',scanAuto:'Discover IPs',updateAction:'Update',settings:'Settings',geoDirectory:'Geo rules directory',geoAuto:'Automatic updates',geoipUpdate:'GeoIP',geositeUpdate:'GeoSite',disabled:'Disabled',daily:'Daily',weekly:'Weekly',directTitle:'Direct addresses and domains',directIps:'Direct IP addresses',rebindTitle:'DNS Rebind exceptions',rebindDomains:'Domains',remove:'Remove',add:'Add',backendRequired:'Install the new Pol package to use this section.',advanced:'Advanced settings'});

Object.assign(words.fa, {
 dns:'DNS راه‌دور مشترک', localDns:'DNS محلی', ssh:'SSH', sshHost:'سرور SSH', sshUser:'نام کاربری',
 sshHostKey:'فینگرپرینت SHA256 سرور', sshPassword:'رمز عبور', sshPrivateKey:'کلید خصوصی',
 sshPassphrase:'رمز کلید خصوصی', sshKeepAlive:'Keepalive SSH (ثانیه)', sshTCPKeepAlive:'Keepalive TCP (ثانیه)',
 sshHelp:'اتصال SSH فقط TCP است. فینگرپرینت کلید میزبان را از مدیر سرور بگیرید.',
 refreshWorker:'به‌روزرسانی اسکریپت Worker', workerTokenRequired:'توکن API را در فرم زیر وارد کنید.',
 'snowflake-amp':'Snowflake AMP', configView:'کانفیگ Xray', configDownload:'دریافت کانفیگ', logSearch:'جست‌وجو در گزارش‌ها', noLogResults:'نتیجه‌ای پیدا نشد.'
});
Object.assign(words.en, {
 dns:'Global remote DNS', localDns:'Local DNS', ssh:'SSH', sshHost:'SSH server', sshUser:'Username',
 sshHostKey:'Server SHA256 fingerprint', sshPassword:'Password', sshPrivateKey:'Private key',
 sshPassphrase:'Private key passphrase', sshKeepAlive:'SSH keepalive (seconds)', sshTCPKeepAlive:'TCP keepalive (seconds)',
 sshHelp:'SSH carries TCP traffic. Get the host key fingerprint from your server administrator.',
 refreshWorker:'Update Worker script', workerTokenRequired:'Enter the API token in the form below.',
 'snowflake-amp':'Snowflake AMP', configView:'Xray configuration', configDownload:'Download configuration', logSearch:'Search logs', noLogResults:'No matching entries.'
});

Object.assign(words.fa,{
 cfSignIn:'ورود به حساب کلادفلر', cfSignOut:'خروج از حساب',
 cfSignedIn:'دسترسی به حساب کلادفلر تأیید شد.',
 cfSignInHelp:'اکانت آی‌دی فعلی و API Token را وارد کنید. توکن فقط تا زمانی که این صفحه باز است، در حافظه مرورگر می‌ماند.',
 cfSignInRequired:'ابتدا با توکن حساب این استقرار و نوع سرویس آن وارد شوید.',
 cfDeployHelp:'برای ساخت یا مدیریت استقرار، توکن باید مجوز ویرایش سرویس انتخاب‌شده را داشته باشد.'
});
Object.assign(words.en,{
 cfSignIn:'Sign in to Cloudflare', cfSignOut:'Sign out',
 cfSignedIn:'Cloudflare account access verified.',
 cfSignInHelp:'Enter your existing Account ID and API token. The token stays in browser memory while this page is open.',
 cfSignInRequired:'Sign in with a token for this deployment’s account and product first.',
 cfDeployHelp:'Creating or managing deployments requires edit access to the selected product.'
});

Object.assign(words.fa,{
 workerDomain:'دامنهٔ سفارشی اتصال', cfDomain:'دامنهٔ سفارشی کلادفلر', cfUsage:'مصرف ۲۴ ساعت گذشته', savedChains:'زنجیره‌های ذخیره‌شده', selectedChain:'زنجیره برای این اتصال', lanSharing:'اشتراک پراکسی در LAN',
 lanUsername:'نام کاربری پراکسی LAN',lanPassword:'رمز پراکسی LAN',
 serverlessDnsType:'پروتکل DNS بدون سرور',serverlessDnsServers:'سرورهای DNS بدون سرور',
 autoWorkerOn:'فعال‌کردن به‌روزرسانی خودکار Worker',autoWorkerOff:'غیرفعال‌کردن به‌روزرسانی خودکار Worker',
 autoWorkerHelp:'برای به‌روزرسانی پس از نصب نسخه‌های بعدی، توکن این حساب با دسترسی فقط روت روی روتر ذخیره می‌شود.'
});
Object.assign(words.en,{
 workerDomain:'Custom connection domain', cfDomain:'Cloudflare custom domain', cfUsage:'Usage over the last 24 hours', savedChains:'Saved proxy chains', selectedChain:'Chain for this connection', lanSharing:'Share LAN proxy',
 lanUsername:'LAN proxy username',lanPassword:'LAN proxy password',
 serverlessDnsType:'Serverless DNS protocol',serverlessDnsServers:'Serverless DNS servers',
 autoWorkerOn:'Enable automatic Worker updates',autoWorkerOff:'Disable automatic Worker updates',
 autoWorkerHelp:'The router stores this account token in a root-only file for updates after future upgrades.'
});

return view.extend({
	handleSave:null, handleSaveApply:null, handleReset:null,
	loadStyle:function(){var link=document.querySelector('link[data-pol-style]');if(link&&link.sheet)return Promise.resolve();return new Promise(function(resolve,reject){if(!link){link=node('link',{'rel':'stylesheet','href':L.resource('pol/style.css')+'?v=5.9.1','data-pol-style':'1'});document.head.appendChild(link);}link.addEventListener('load',resolve,{once:true});link.addEventListener('error',function(){reject(new Error('Unable to load Pol stylesheet'));},{once:true});});},
 load:function() { return Promise.all([api.settings(),api.status(),this.loadStyle()]); },
	render:function(data) {
		var self=this;
		this.s=data[0]; this.status=data[1]; this.lang=localStorage.getItem('pol-language')||'fa'; this.theme=localStorage.getItem('pol-theme')||'auto'; this.tab='home'; this.logKind='all'; this.busy=false; this.queue=Promise.resolve();
		if (!words[this.lang]) this.lang='fa';
		if (['auto','light','dark'].indexOf(this.theme)<0) this.theme='auto';
		this.media=window.matchMedia('(prefers-color-scheme: dark)');
		this.media.addEventListener('change',function(){if(self.theme==='auto')self.draw();});
		if (this.s.error) return node('div',{'class':'alert-message warning'},this.s.error);
		if (!document.querySelector('link[data-pol-style]')) document.head.appendChild(node('link',{'rel':'stylesheet','href':L.resource('pol/style.css')+'?v=5.9.1','data-pol-style':'1'}));
		this.root=node('div');this.root.addEventListener('input',function(ev){self.fitFields(ev.target);});this.root.addEventListener('change',function(ev){self.fitFields(ev.target);}); this.draw();if(document.fonts)document.fonts.ready.then(function(){self.fitFields();});
 var closeHelp=function(ev){if(self.activeHelp&&(!ev||!self.activeHelp.contains(ev.target))){self.activeHelp.open=false;self.activeHelp.querySelector('summary').setAttribute('aria-expanded','false');self.activeHelp=null;}};document.addEventListener('click',closeHelp);document.addEventListener('scroll',closeHelp,true);window.addEventListener('resize',closeHelp);document.addEventListener('keydown',function(ev){if(ev.key==='Escape'&&self.activeHelp){var summary=self.activeHelp.querySelector('summary');closeHelp();summary.focus();}});
		poll.add(function(){ return api.status().then(function(v){self.status=v;if(self.flash&&self.flash.dataset.rpcFault==='true')self.message(''); self.paintStatus();self.paintGeoProgress();self.paintScanFeedback(); if(self.tab==='reports'&&!self.pauseLogs) return self.fetchLogs();}).catch(function(e){self.message(e.message,true);}); },5);
		return this.root;
	},

 help:function(key,text){
  if(['core','state','connectionStatus','main','usage','connectionCheck'].indexOf(key)>=0)return null;
  var self=this, body=text||this.helpText(key);if(!body)return null;
  var box=node('div',{'class':'pol-help-content','role':'note'},body),summary=node('summary',{'aria-label':this.t('help')+' · '+this.t(key),'aria-expanded':'false'},this.lang==='fa'?'؟':'?'),details=node('details',{'class':'pol-help'},[summary,box]);
  summary.addEventListener('click',function(ev){ev.preventDefault();ev.stopPropagation();var opening=!details.open;if(self.activeHelp){self.activeHelp.open=false;self.activeHelp.querySelector('summary').setAttribute('aria-expanded','false');}self.activeHelp=null;if(opening){details.open=true;self.activeHelp=details;summary.setAttribute('aria-expanded','true');var a=summary.getBoundingClientRect(),b=box.getBoundingClientRect();box.style.left=Math.max(16,Math.min(self.lang==='fa'?a.right-b.width:a.left,window.innerWidth-b.width-16))+'px';box.style.top=Math.max(16,a.bottom+8+b.height<window.innerHeight-16?a.bottom+8:a.top-b.height-8)+'px';}});
  return details;
 },
 helpText:function(key){
  if(key==='localDns')return this.lang==='fa'?'آدرس IPv4 یا IPv6 برای نام دامنه‌های مسیر مستقیم.':'IPv4 or IPv6 resolver for directly routed domains.';
  var fa=this.lang==='fa',fragmentKeys=['fragments','masqueFragments','serverlessFragments'],noiseKeys=['udpNoises','masqueNoises','serverlessNoises'];
  if(fragmentKeys.indexOf(key)>=0){var scope={fragments:fa?'Worker؛ در نبود قانون اختصاصی، برای MASQUE و بدون سرور هم استفاده می‌شود':'Worker; also the fallback for MASQUE and serverless',masqueFragments:'MASQUE',serverlessFragments:fa?'بدون سرور':'Serverless'}[key];return (fa?'قوانین خردکردن بسته‌های TCP برای '+scope+'. فقط وقتی روش انتخاب‌شده از فرگمنت استفاده کند اعمال می‌شوند. یک آرایه JSON وارد کنید؛ [] یعنی استفاده از مقادیر ساده یا قوانین عمومی. packets بسته‌ها را انتخاب می‌کند: tlshello یا 1-1، 1-2، 1-3، 1-5. lengths فهرست بازه‌های طول به بایت و delays فهرست بازه‌های تأخیر به میلی‌ثانیه است. هر بازه min و max صعودی بین ۰ تا ۶۵۵۳۵ دارد. maxSplit اختیاری است و تعداد تکه‌ها را محدود می‌کند. حداکثر ۳۲ قانون و ۳۲ بازه در هر فهرست. نمونه: ':'TCP fragmentation rules for '+scope+'. Applied only when the chosen transport uses fragmentation. Enter a JSON array; [] uses simple settings or shared rules. packets: tlshello, 1-1, 1-2, 1-3 or 1-5. lengths: byte ranges; delays: millisecond ranges. Each range has min <= max, from 0 to 65535. Optional maxSplit limits split count. Maximum 32 rules and 32 ranges per list. Example: ')+JSON.stringify([{packets:'tlshello',lengths:[{min:100,max:200}],delays:[{min:1,max:3}]}]);}
  if(noiseKeys.indexOf(key)>=0){var scope={udpNoises:fa?'WARP؛ در نبود قانون اختصاصی، برای MASQUE و بدون سرور هم استفاده می‌شود':'WARP; also the fallback for MASQUE and serverless',masqueNoises:'MASQUE',serverlessNoises:fa?'بدون سرور':'Serverless'}[key];return (fa?'قوانین تولید نویز UDP برای '+scope+'. فقط در مسیرهایی که نویز UDP دارند اعمال می‌شود؛ اتصال TCP از این نویز استفاده نمی‌کند. آرایه JSON وارد کنید؛ [] از مقادیر ساده یا قوانین عمومی استفاده می‌کند. type یکی از rand (بایت تصادفی)، str (متن)، hex (هگز)، base64، array (آرایه بایت) یا exp (الگوی هسته) است. size بازه طول برای rand به بایت، delay بازه تأخیر به میلی‌ثانیه و count تعداد تکرار ۱ تا ۱۰۰ است؛ صفر معادل یک است. برای array از bytes با اعداد ۰ تا ۲۵۵ و برای سایر انواع از packet استفاده کنید. همه بازه‌ها min و max صعودی بین ۰ تا ۶۵۵۳۵ دارند. مجموع تکرارها حداکثر ۲۵۶ و طول داده حداکثر ۴۰۹۶ است. نمونه: ':'UDP noise rules for '+scope+'. Used only by transports with UDP noise, not TCP. Enter a JSON array; [] uses simple settings or shared rules. type: rand, str, hex, base64, array or exp (core expression). size is the random payload byte range; delay is milliseconds; count is 1–100 (0 means 1). array uses bytes (0–255); other non-random types use packet. Ranges require 0 <= min <= max <= 65535. Total repeats <=256; payload length <=4096. Example: ')+JSON.stringify([{type:'rand',size:{min:10,max:30},count:1,delay:{min:1,max:3}}]);}
  var extra=fa?{fakeSniDns:'در حالت بدون سرور، روش DNS راه‌دور، درخواست‌های DoH را با SNI جایگزین www.google.com به Cloudflare، Google و Quad9 می‌فرستد؛ نام معتبر گواهی هر سرویس جداگانه بررسی می‌شود. فقط در همین روش اثر دارد و روی Worker، WARP، MASQUE و کانفیگ واردشده اعمال نمی‌شود. ممکن است برخی شبکه‌ها یا سرورها این روش را نپذیرند.',fakeDns:'به دامنه‌ها IP مجازی می‌دهد تا هسته مقصد اصلی را بازیابی و مسیریابی کند. این IPها آدرس واقعی سایت نیستند. روی کانفیگ واردشده اعمال نمی‌شود.',tcpFastOpen:'امکان ارسال داده در آغاز اتصال TCP را فعال می‌کند تا در شبکه‌های سازگار تأخیر شروع کمتر شود. پشتیبانی هسته و مسیر شبکه لازم است؛ تضمینی برای بهبود همه اتصال‌ها نیست.',warpReserved:'بایت‌های Reserved ثبت‌شده در اکانت WARP را به هدر WireGuard اضافه می‌کند. فقط در مسیر WARP اثر دارد؛ خاموش‌کردن اطلاعات اکانت را حذف نمی‌کند.',workerProxyIps:'فهرست اختیاری مقصدهای جایگزین برای اتصال TCP داخل ورکر. اگر اتصال مستقیم ورکر به مقصد برقرار نشود، ورکر یکی از این آدرس‌ها را امتحان می‌کند. هر IP یا دامنه را جداگانه اضافه کنید؛ این گزینه با IP تمیز اتصال روتر به کلادفلر متفاوت است. هنگام ساخت ورکر در آن ذخیره می‌شود؛ تغییر این فهرست ورکر قبلی را خودکار به‌روزرسانی نمی‌کند.',workerFallback:'اختیاری؛ آدرس کامل یک سایت مانند https://example.com را وارد کنید. درخواست‌های عادی HTTP که به تونل تعلق ندارند، محتوای این سایت را از طریق ورکر دریافت می‌کنند. پروتکل، دامنه و پورت از این نشانی گرفته می‌شود؛ مسیر و پارامترهای درخواست اصلی حفظ می‌شوند. خالی باشد یا دریافت شکست بخورد، پاسخ 404 نمایش داده می‌شود. این گزینه سرور پشتیبان VPN نیست و هنگام ساخت ورکر ذخیره می‌شود.'}:{fakeSniDns:'Only for Serverless > Remote DNS. Sends DoH to Cloudflare, Google and Quad9 with www.google.com as alternate SNI, while separately verifying each resolver certificate identity. Does not affect Worker, WARP, MASQUE or imported configurations. Network and server support varies.',fakeDns:'Maps domains to virtual IPs so the core can recover the destination and route it. These are not real destination IPs. Does not modify imported configurations.',tcpFastOpen:'Enables early TCP data where the kernel and network support it. It may reduce connection setup latency; improvement is not guaranteed.',warpReserved:'Adds Reserved bytes from the registered WARP account to WireGuard headers. Only affects WARP; disabling does not delete the account.',workerProxyIps:'Optional TCP retry destinations inside the deployed Worker, used if its direct destination connection fails. Add each IP or hostname separately. These are not clean IPs used by the router to reach Cloudflare. Saved at deployment; changing this list does not automatically update an existing Worker.',workerFallback:'Optional full URL such as https://example.com. Ordinary non-tunnel HTTP requests fetch this site through the Worker. Its protocol, hostname and port replace those of the request; the original request path and query remain. Empty or failed fetch returns 404. This is not a backup VPN server. Saved at deployment.'};
  if(extra[key])return extra[key];
  var hints=this.lang==='fa'?{
   connection:'نوع اتصال و روش متناسب با شبکه را انتخاب کنید. هر تغییر بلافاصله ذخیره می‌شود و در حالت روشن ممکن است اتصال را راه‌اندازی مجدد کند.', mode:'ورکر به دامنه و UUID نیاز دارد؛ WARP و MASQUE حساب جدا می‌خواهند. حالت کانفیگ واردشده از لینک یا JSON استفاده می‌کند.', variant:'روش معمولی را ابتدا امتحان کنید. فرگمنت برای TCP و مبهم‌سازی برای UDP است. برای Worker فرگمنت ابتدا ECH را خاموش کنید.', host:'فقط دامنه Worker را بدون https:// و بدون مسیر وارد کنید.', uuid:'شناسه محرمانه همان Worker؛ بعد از ساخت خودکار پر می‌شود.', address:'اختیاری؛ IP تمیز یا دامنه مقصد. خالی باشد از دامنه Worker استفاده می‌شود.', path:'مسیر WebSocket همان Worker؛ پیش‌فرض /?ed=2560 است.', workerSni:'نام سرور TLS؛ خالی باشد دامنه Worker استفاده می‌شود. گواهی مقصد باید معتبر باشد.', port:'پورت اتصال TLS؛ معمولا 443.', fragmentLength:'طول هر تکه، به بایت؛ عدد یا بازه صعودی مثل 100-200.', fragmentDelay:'فاصله زمانی تکه‌ها به میلی‌ثانیه؛ عدد یا بازه صعودی مثل 1-3.', dns:'آدرس IPv4 یا IPv6 را وارد کنید.', lan:'نام رابط واقعی شبکه LAN، معمولا br-lan؛ فقط ترافیک همین رابط گرفته می‌شود.', transparent:'با روشن‌بودن، دستگاه‌های LAN از پل استفاده می‌کنند؛ خاموش باشد پروکسی فقط روی خود روتر است.', endpoint:'نقطه اتصال به صورت IP:port یا [IPv6]:port؛ مقدار مناسب را از اسکن انتخاب کنید.', transport:'h2 از TCP/TLS و h3 از UDP/QUIC استفاده می‌کند. مسدودبودن UDP می‌تواند h3 را مختل کند.', sni:'نام سرور TLS سرویس MASQUE.', chain:'پروکسی دوم با URI؛ مسیر دستگاه ← اتصال اصلی ← پروکسی دوم ← اینترنت می‌شود. خالی باشد زنجیره خاموش است.', logLevel:'برای بررسی موقت Debug و برای استفاده روزمره Warning مناسب است.', tlsFingerprint:'ظاهر دست‌دهی TLS. روی Worker و MASQUE h2 اعمال می‌شود.', advanced:'این گزینه‌ها روی اتصال‌های ساخته‌شده در پل اعمال می‌شوند؛ خروجی‌های JSON تنظیمات خود را دارند.', echDns:'IP یا نشانی HTTPS سرویس DoH؛ فقط Worker معمولی. هم‌زمان با فرگمنت فعال نمی‌شود.', tcpFastOpen:'ارسال زودتر داده در اتصال TCP ورکر؛ به پشتیبانی هسته و شبکه بستگی دارد.', dnsStrategy:'فقط IPv4، فقط IPv6 یا هر دو. در شبکه فاقد IPv6 حالت فقط IPv6 را انتخاب نکنید.', noiseSize:'اندازه نویز تصادفی UDP به بایت؛ عدد یا بازه صعودی.', noiseDelay:'فاصله ارسال نویز UDP، به میلی‌ثانیه.', noiseReset:'بازه بازنشانی نویز UDP در هسته.', warpMtu:'اندازه MTU وارپ، بین 1280 و 1500.', warpKeepAlive:'فاصله Keepalive وارپ به ثانیه؛ صفر آن را غیرفعال می‌کند.', warpReserved:'استفاده از بایت‌های Reserved حساب وارپ؛ خاموش باشد سه بایت صفر فرستاده می‌شود.', account:'ثبت حساب به دسترسی مستقیم روتر به Cloudflare نیاز دارد. حساب‌های WARP و MASQUE جدا هستند.', cfTitle:'در Cloudflare زیر‌دامنه workers.dev را فعال و یک API Token با مجوز ساخت Worker برای همان حساب بسازید. Token و Account ID متفاوت‌اند. توکن در تنظیمات ذخیره نمی‌شود.', routing:'دامنه‌های مستقیم و مسدود، هرکدام در یک خط. تفکیک ایران در صورت فعال‌بودن پیش از این فهرست‌ها انجام می‌شود.', routeIran:'GeoIP ایران در فایروال پیش از ورود به هسته مستقیم می‌رود. دامنه‌های GeoSite با DNS مستقیم و مجموعه nftset از تونل خارج می‌شوند؛ قواعد ایران پیش از فیلترینگ پل قرار دارند. ابتدا هر دو فایل را دریافت کنید. برای این روش dnsmasq با پشتیبانی nftset لازم است.', iranDns:'IPv4 سرویس DNS مستقیم برای دامنه‌های ایران؛ این DNS از تونل عبور نمی‌کند.', geoTitle:'GeoIP و GeoSite جداگانه دریافت و بررسی می‌شوند. فایل باید گروه IR داشته باشد. اگر دریافت یا فعال‌سازی شکست بخورد فایل قبلی حفظ می‌شود.', usage:'ارسال، دریافت و مجموع واقعی ترافیک عبوری از ورودی‌های پل، بر حسب GB اعشاری. هفته و ماه به معنی ۷ و ۳۰ روز اخیر هستند. ترافیک ایران که پیش از هسته مستقیم می‌رود شمرده نمی‌شود. آمار از زمان نصب این نسخه جمع می‌شود، هر ۱۵ ثانیه نمونه‌برداری و هر ۵ دقیقه یا خاموش‌کردن ذخیره می‌شود.', connectionStatus:'وضعیت اینترنت با درخواست HTTPS از داخل اتصال پل بررسی می‌شود؛ اجرای هسته به‌تنهایی به معنی اتصال اینترنت نیست.', core:'نسخه باینری هسته و وضعیت اجرای پردازش آن روی روتر.', server:'آدرس اتصال انتخاب‌شده و IP و کشور خروجی واقعی اتصال. پینگ زمان درخواست HTTPS است. پرچم فقط پس از دریافت کشور واقعی نمایش داده می‌شود.', main:'کلید اصلی، همه پردازش‌های اتصال پل و بررسی‌های آن را روشن یا خاموش می‌کند. هنگام خاموش‌کردن، اسکن یا عملیات شبکه در حال اجرا نیز لغو می‌شود.', export:'خروجی شامل کلیدها و UUID است؛ آن را محرمانه نگه دارید. برای بازیابی، دکمهٔ بازیابی را بزنید و فایل پشتیبان پل را انتخاب کنید.', license:'برنامه و Worker: GPL-3.0-only؛ ظاهر کارت‌ها: AGPL-3.0-or-later؛ Xray: MPL-2.0؛ وزیرمتن: OFL-1.1. جزییات و منابع در NOTICE مخزن هستند.'
  }:{connection:'Choose a connection and variant. Changes save immediately and can restart an active tunnel.',routeIran:'Bypass Iranian IPs before TProxy. A dedicated split DNS resolver maps GeoSite domains to nftsets, before filtering. Download both files first; dnsmasq must support nftset.',geoTitle:'Update GeoIP and GeoSite separately. Files must contain usable IR entries. Failed downloads keep the previous file.',usage:'Actual Xray inbound traffic in decimal GB. Week and month cover the last 7 and 30 days. Pre-routed direct traffic is excluded. Checkpoints every five minutes and on stop.',connectionStatus:'An HTTPS request through the tunnel checks internet connectivity. Core process state alone does not establish connectivity.',core:'Installed core version and process state.',server:'Configured server and verified exit IP, country and HTTPS request latency.',main:'Controls all tunnel processes and connectivity checks. Turning off also cancels ongoing network jobs.',export:this.t('exportHelp'),advanced:this.t('advancedHelp'),cfTitle:this.t('cfHelp'),account:this.t('registrationHelp'),echDns:this.t('echHelp'),routing:this.t('domainHelp'),license:'Pol/Worker: GPL-3.0-only; dashboard styles: AGPL-3.0-or-later; Xray: MPL-2.0; Vazirmatn: OFL-1.1. See NOTICE in the repository.'};
  return hints[key]||null;
 },
 label:function(key,text){return node('span',{'class':'pol-label'},[node('span',{},this.t(key)),this.help(key,text)]);},
 dashboardCards:function(){
  var self=this,s=this.status,c=s.connection||{},usage=s.usage||{},period=this.usagePeriod||'today',totals=usage[period]||{up:0,down:0},connected=!!(s.running&&c.connected);
  function row(key,value,attrs){return node('div',{'class':'pol-dashboard-row'},[node('span',{},self.t(key)),node('span',attrs||{},value)]);}
  function metricIcon(name){return node('span',{'class':'pol-metric-icon'},icon(name));}
  var country=connected&&/^[A-Z]{2}$/.test(c.country||'')?c.country:'';
  if(!this.lastFlagCountry){try{var savedCountry=localStorage.getItem('pol-last-flag-country');if(/^[A-Z]{2}$/.test(savedCountry||''))this.lastFlagCountry=savedCountry;}catch(e){}}
  var flagOrb=node('div',{'class':'pol-country-orb'},this.lastFlagCountry?node('img',{'src':'https://flagcdn.com/w320/'+this.lastFlagCountry.toLowerCase()+'.png','alt':'','referrerpolicy':'no-referrer'}):icon('globe'));
  if(country&&country!==this.lastFlagCountry){
   var nextFlag=node('img',{'alt':'','referrerpolicy':'no-referrer'});
   nextFlag.onload=function(){self.lastFlagCountry=country;try{localStorage.setItem('pol-last-flag-country',country);}catch(e){}dom.content(flagOrb,nextFlag);};
   nextFlag.src='https://flagcdn.com/w320/'+country.toLowerCase()+'.png';
  }
  var chosenExit=this.s.mode==='custom'?'none':this.s.mode==='tor'||this.s.mode==='psiphon'?this.s.mode:this.s.exitMode,exit=s.exit&&s.exit.engine===chosenExit?s.exit:null,exitText='';if(s.enabled&&exit&&exit.phase!=='stopped'&&(exit.phase!=='ready'||(!connected&&!c.checked))){exitText=exit.phase==='ready'?(this.lang==='fa'?'موتور خروجی آماده است؛ وضعیت اینترنت جداگانه بررسی می‌شود.':'Exit engine ready; internet is checked separately.'):exit.phase.indexOf('bootstrap:')===0?'Tor · '+exit.phase.split(':')[1]+'%':this.lang==='fa'?(exit.phase==='reconnecting'?'در حال اتصال مجدد موتور خروجی':exit.phase==='fetching-bridges'?'در حال دریافت پل‌های Tor':exit.phase==='bridge-fetch-failed'?'دریافت پل‌های Tor ناموفق بود':'در حال راه‌اندازی موتور خروجی'):exit.phase;}
  var state=this.t(!s.running?'off':connected?'internetOk':exit&&!exit.ready?'internetWaiting':c.checked?'internetFailed':'internetWaiting');
  var serverCard=node('section',{'class':'pol-dashboard-card pol-dashboard-server pol-server-card','data-dashboard':'server','data-connected':String(connected),'style':'--accent:#10b981'},[
   node('h3',{'class':'pol-dashboard-card-title'},[node('span',{'class':'pol-card-icon'},icon('pulse')),this.label('state',this.helpText('server')),node('span',{'class':'pol-server-state'},state)]),
   node('div',{},[row('nodeName',this.t(s.mode||this.s.mode)),exitText?node('small',{'role':'status'},exitText):null,row('server',s.server||'—',{'dir':'ltr','class':'pol-server-private pol-server-host','tabindex':'0'}),row('ping',connected?c.latency+(this.lang==='fa'?' م‌ثانیه':' ms'):'—'),row('exitIp',connected?c.exitIp||'—':'—',{'dir':'ltr','class':'pol-server-private pol-server-exit','tabindex':'0'})]),
   node('div',{'class':'pol-server-art','aria-hidden':'true'},[node('div',{'class':'pol-orbit pol-orbit-one'}),node('div',{'class':'pol-orbit pol-orbit-two'}),flagOrb])
  ]);
  var version=(s.coreVersion||'').match(/^Xray\s+([^\s]+)/);
  var coreCard=node('section',{'class':'pol-dashboard-card pol-core-tile','data-dashboard':'core','style':'--accent:#3b82f6'},[
   metricIcon('connection'),node('div',{'class':'pol-core-title'},[this.label('core'),node('span',{},' Xray'),node('div',{'class':'pol-core-subversion'},version?version[1]:s.coreVersion||'—')]),node('div',{'class':'pol-core-state '+(s.running?'pol-core-on':'pol-core-off')},this.t(s.running?'coreRunning':'coreStopped'))
  ]);
  var checkCard=node('section',{'class':'pol-dashboard-card pol-dashboard-checks','data-dashboard':'checks'},[
   node('div',{'class':'pol-usage-title'},[metricIcon('globe'),this.label('connectionCheck',this.helpText('connectionStatus'))])
  ].concat([['telegram','Telegram','#26a5e4','telegram'],['youtube','YouTube','#ef4444','video'],['google','Google','#4285f4','globe']].map(function(site){
   var result=(self.siteChecks||{})[site[0]],value=!s.running?'—':result?(result.ok?result.latency+(self.lang==='fa'?' م‌ثانیه':' ms'):self.t('failed')):self.t('checkNow');
   var button=node('button',{'type':'button','class':'pol-site-check','style':'--site-colour:'+site[2],'click':function(){self.action(function(){return api.check(site[0]);}).then(function(r){if(r&&r.site){self.siteChecks=self.siteChecks||{};self.siteChecks[r.site]=r;self.paintStatus();}});}},[node('span',{'class':'pol-site-icon'},icon(site[3])),node('span',{'class':'pol-site-name'},site[1]),node('span',{'class':'pol-site-value'},value)]);button.dataset.polManaged='site';button.disabled=self.busy||!s.running||!(s.features&&s.features.siteChecks);return button;
  })));
  var usageCard=node('section',{'class':'pol-dashboard-card pol-dashboard-usage','data-dashboard':'usage'},[
   node('div',{'class':'pol-usage-title'},[metricIcon('reports'),this.label('usage')]),
   node('div',{'class':'pol-usage-body'},[
    node('div',{'class':'pol-usage-values'},[['upload',totals.up,'up'],['downloadBytes',totals.down,'down'],['total',Number(totals.up)+Number(totals.down),'total']].map(function(v){return node('div',{'class':'pol-usage-value pol-usage-'+v[2]},[node('span',{},self.t(v[0])),node('strong',{'dir':'ltr'},(Math.max(0,Number(v[1])||0)/1000000000).toFixed(2))]);})),
    node('div',{'class':'pol-usage-periods','role':'tablist','aria-label':this.t('usage')},['today','week','month'].map(function(k,i){return node('button',{'type':'button','role':'tab','aria-selected':String(period===k),'title':self.t(k),'click':function(){self.usagePeriod=k;self.paintStatus();}},self.lang==='fa'?['۱ر','۷ر','۳۰ر'][i]:['1d','1w','1m'][i]);}))
   ])
  ]);
  return [serverCard,coreCard,checkCard,usageCard];
 },
 fitFields:function(target){
  if(!this.root)return;if(!this.root.isConnected){var self=this;if(!this.fieldFitScheduled){this.fieldFitScheduled=true;requestAnimationFrame(function(){self.fieldFitScheduled=false;if(self.root&&self.root.isConnected)self.fitFields();});}return;}var canvas=this.fieldCanvas||(this.fieldCanvas=document.createElement('canvas')),context=canvas.getContext('2d');
  var fields=target&&target.matches&&target.matches('.pol-input')?[target]:this.root.querySelectorAll('.pol-input');
  Array.prototype.forEach.call(fields,function(el){
   if(el.closest('.pol-grid,.pol-log-tools'))return;
   var style=window.getComputedStyle(el);context.font=(style.fontWeight||'400')+' '+(style.fontSize||'14px')+' '+(style.fontFamily||'Vazirmatn');
   var value=el.tagName==='SELECT'?(el.options[el.selectedIndex]||{}).text||'':el.value||el.placeholder||'';
   if(el.type==='password'&&el.value)value='•'.repeat(el.value.length);
   var lines=value.split('\n'),width=0;lines.forEach(function(line){width=Math.max(width,context.measureText(line).width);});
   var extra=(parseFloat(style.paddingLeft)||13)+(parseFloat(style.paddingRight)||13)+4+(el.tagName==='SELECT'?32:el.type==='number'?20:8);
   el.style.width=Math.ceil(Math.max(el.tagName==='TEXTAREA'?240:88,width+extra))+'px';
  });
 },
 iranDNSField:function(){
  var self=this,fa=this.lang==='fa',options=[['178.22.122.100',fa?'شکن · 178.22.122.100':'Shecan · 178.22.122.100'],['1.1.1.1','Cloudflare · 1.1.1.1'],['8.8.8.8','Google · 8.8.8.8'],['9.9.9.9','Quad9 · 9.9.9.9']],custom=this.iranDNSCustom||!options.some(function(v){return v[0]===self.s.iranDns;});
  var select=node('select',{'class':'pol-input','aria-label':this.t('iranDns'),'change':function(ev){if(ev.target.value==='custom'){self.iranDNSCustom=true;self.panel();}else{self.iranDNSCustom=false;self.save('iranDns',ev.target.value).then(function(){self.panel();});}}},options.map(function(v){return node('option',{'value':v[0]},v[1]);}).concat([node('option',{'value':'custom'},this.t('customSource'))]));select.value=custom?'custom':this.s.iranDns;
  return node('div',{'class':'pol-iran-dns'},[node('label',{'class':'pol-field'},[this.label('iranDns',fa?'فقط هنگام فعال بودن تفکیک ایران استفاده می‌شود. دامنه‌های GeoSite ایران از این DNS و خارج از تونل پرسیده می‌شوند؛ پاسخ‌ها برای مسیریابی مستقیم در nftset ثبت می‌شوند. GeoIP برای IPهای صریح به این DNS نیاز ندارد. سرویس انتخابی باید از اینترنت مستقیم روتر در دسترس باشد؛ این درخواست‌ها رمزگذاری نمی‌شوند.':'Used only when Iran bypass is enabled. Iranian GeoSite domains resolve outside the tunnel and their addresses populate direct-routing nftsets. Literal GeoIP rules do not require this resolver. It must be reachable directly; these DNS queries are unencrypted.'),select]),custom?this.field('iranDns',null,'text'):null]);
 },
 dnsProbeError:function(text){
  if(this.lang!=='fa')return text;
  var errors={'invalid DNS probe request':'اطلاعات آزمایش DNS معتبر نیست.','invalid DNS probe server':'نوع یا نام سرور DNS معتبر نیست.','invalid DNS probe IP':'آدرس IP سرور DNS معتبر نیست.','UDP DNS probe requires an IP address':'برای آزمایش UDP، آدرس IP سرور را وارد کنید؛ مانند 8.8.8.8.','another background operation is in progress':'عملیات دیگری در حال اجراست؛ پس از پایان آن دوباره امتحان کنید.','another operation is in progress':'عملیات دیگری در حال اجراست؛ پس از پایان آن دوباره امتحان کنید.','Operation timed out; check Logs.':'مهلت آزمایش DNS پایان یافت؛ گزارش‌ها را بررسی کنید.'};
  return errors[text]||'آزمایش DNS انجام نشد؛ دسترسی اینترنت روتر و آدرس سرور را بررسی کنید.';
 },
 paintDNSFeedback:function(){if(!this.dnsFeedbackNode)return;var f=this.dnsProbeFeedback;this.dnsFeedbackNode.textContent=f?f.text:'';this.dnsFeedbackNode.className='pol-dns-feedback'+(f&&f.error?' is-error':'');this.dnsFeedbackNode.setAttribute('role',f&&f.error?'alert':'status');},
 dnsProbePanel:function(){
  var self=this,fa=this.lang==='fa',saved=this.dnsProbeRequest||{type:'udp',host:'dns.google',ips:'8.8.8.8'};
  var type=node('select',{'class':'pol-input','aria-label':fa?'روش آزمایش':'Test protocol'},['udp','tcp','doh','doq'].map(function(t){return node('option',{'value':t},t.toUpperCase());})),host=node('input',{'class':'pol-input','dir':'ltr','value':saved.host,'placeholder':'dns.google'}),ips=node('input',{'class':'pol-input','dir':'ltr','value':saved.ips,'placeholder':'8.8.8.8'});type.value=saved.type;
  function field(text,input){return node('label',{'class':'pol-field'},[node('span',{},text),input]);}
  var button=node('button',{'type':'button','class':'pol-btn','click':function(){if(self.busy)return;self.dnsProbeRequest={type:type.value,host:host.value.trim(),ips:ips.value.trim()};self.action(function(){return api.dnsProbe({type:type.value,host:host.value.trim(),ips:ips.value.split(/[ ,]+/).filter(Boolean)});},null,'dns').then(function(r){if(!r||typeof r.ok!=='boolean')return;self.dnsProbeFeedback={text:r.ok?(fa?'پاسخ معتبر، زمان: ':'Valid response, latency: ')+Math.round(r.latency||0)+(fa?' م‌ثانیه':' ms'):(fa?'آزمایش ناموفق بود؛ این DNS از اینترنت مستقیم روتر پاسخ معتبر نداد.':'Test failed; this DNS did not return a valid response over the router direct connection.'),error:!r.ok};self.paintDNSFeedback();});}},fa?'آزمایش DNS':'Test DNS');
  this.dnsFeedbackNode=node('div',{'aria-live':'polite'});this.paintDNSFeedback();
  return node('section',{'class':'pol-dns-probe'},[node('strong',{},fa?'آزمایش DNS مستقیم روتر':'Router direct DNS test'),node('div',{'class':'pol-grid'},[field(fa?'روش آزمایش':'Test protocol',type),field(fa?'نام سرور DNS':'DNS hostname',host),field(fa?'آدرس‌های IP سرور DNS':'DNS server IPs',ips)]),node('div',{'class':'pol-dns-actions'},[button,this.dnsFeedbackNode])]);
 },
 jsonField:function(key,objectMode){
  var self=this,feedback=node('small',{'role':'alert','style':'color:#dc2626'});
  var input=node('textarea',{'class':'pol-input pol-code','dir':'ltr','rows':6,'aria-label':this.t(key),'change':function(){var value;try{value=JSON.parse(input.value||(objectMode?'{}':'[]'));if(objectMode?(!value||Array.isArray(value)||typeof value!=='object'):!Array.isArray(value))throw new Error('array');}catch(e){feedback.textContent=self.lang==='fa'?'یک فهرست JSON معتبر وارد کنید.':'Enter a valid JSON array.';return;}feedback.textContent='';self.save(key,value);}},JSON.stringify(this.s[key]||(objectMode?{}:[]),null,2));
  return node('div',{'class':'pol-field pol-json-field'},[this.label(key),input,feedback]);
 },
 listField:function(key,title,placeholder,helpText){title=title||key;
  var self=this,supported=!!(this.status.features&&this.status.features.routingLists),list=node('div',{'class':'pol-list-editor'});
  function draw(){
   var values=self.s[key]||[];
   var rows=values.map(function(value,index){return node('div',{'class':'pol-list-item','dir':'ltr'},[node('input',{'class':'pol-input','value':value,'readonly':'readonly','aria-label':self.t(title)}),node('button',{'type':'button','class':'pol-list-remove','disabled':!supported,'aria-label':self.t('remove')+' '+value,'click':function(){var next=(self.s[key]||[]).slice();next.splice(index,1);self.save(key,next).then(draw);}},'×')]);});
   var input=node('input',{'class':'pol-input','dir':'ltr','placeholder':placeholder,'aria-label':self.t(title),'disabled':!supported});
   function add(){var value=input.value.trim();if(!value||self.busy)return;if((self.s[key]||[]).indexOf(value)>=0){input.value='';return;}self.save(key,(self.s[key]||[]).concat([value])).then(draw);}
   input.addEventListener('keydown',function(ev){if(ev.key==='Enter'){ev.preventDefault();add();}});
   rows.push(node('div',{'class':'pol-list-add','dir':'ltr'},[input,node('button',{'type':'button','class':'pol-list-plus','disabled':!supported,'aria-label':self.t('add'),'click':add},'+')]));dom.content(list,rows);if(list.isConnected)self.fitFields();
  }
  draw();return node('div',{'class':'pol-settings-row pol-compact-list'},[this.label(title,helpText),list]);
 },
 paintGeoProgress:function(){
  if(!this.geoProgressNode)return;if(this.geoAwaiting){dom.content(this.geoProgressNode,node('strong',{},this.t('verifying')));return;}var fa=this.lang==='fa',j=this.status.job||{},p=j.progress||{},name=j.target==='geoip'?'GeoIP':j.target==='geosite'?'GeoSite':'',feedback=this.geoFeedback;
  if(j.method!=='geoUpdate'&&!feedback){dom.content(this.geoProgressNode,[]);return;}
  var active=j.method==='geoUpdate'&&(j.state==='running'||j.state==='pending'),failed=j.method==='geoUpdate'&&j.state==='failed',done=j.method==='geoUpdate'&&j.state==='done';
  var phase=active?({connecting:fa?'در حال اتصال به منبع':'Connecting to source',downloading:fa?'در حال دریافت فایل':'Downloading file',verifying:fa?'در حال بررسی فایل':'Verifying file',installing:fa?'در حال نصب و اعمال فایل':'Installing and applying file'}[p.phase]||this.t('verifying')):done?(fa?'دریافت، بررسی و نصب فایل با موفقیت انجام شد.':'File downloaded, verified and installed.'):failed?(fa?'عملیات ناموفق بود؛ فایل قبلی حفظ شد. ':'Update failed; previous file kept. ')+j.error:feedback?feedback.text:'';
  var percent=p.total>0?Math.min(100,Math.floor(p.received*100/p.total)):null,attrs={'max':100,'aria-label':phase};if(percent!==null)attrs.value=percent;
  var nodes=[node('strong',{},(name?name+' · ':'')+phase)];
  if(active){nodes.push(node('progress',attrs));if(p.received>0)nodes.push(node('span',{'dir':'ltr'},(percent!==null?percent+'% · ':'')+(p.received/1048576).toFixed(2)+' MB'+(p.total>0?' / '+(p.total/1048576).toFixed(2)+' MB':'')));}
  if(feedback&&feedback.error&&!failed)nodes.push(node('span',{},feedback.text));
  this.geoProgressNode.classList.toggle('is-error',failed||!!(feedback&&feedback.error));dom.content(this.geoProgressNode,nodes);
 },
 paintGeoReadiness:function(){
  if(!this.geoWarning)return;var self=this,fa=this.lang==='fa',g=this.geo,missing=['geoip','geosite'].filter(function(k){return !g||!g[k]||!g[k].ready;});
  var slider=this.content.querySelector('[role="switch"][aria-label="'+this.t('routeIran')+'"]');
  if(slider)slider.disabled=this.busy||(!this.s.routeIran&&(!g||missing.length>0));
  this.geoWarning.hidden=!!g&&!missing.length;
  this.geoWarning.textContent=!g?(fa?'در حال بررسی فایل‌های مسیریابی…':'Checking routing data…'):missing.length?(fa?'تفکیک ترافیک ایران آماده نیست. فایل‌های '+missing.map(function(k){return k==='geoip'?'GeoIP':'GeoSite';}).join(' و ')+' دریافت نشده‌اند یا معتبر نیستند. ابتدا آن‌ها را از بخش «فایل‌های مسیریابی» دریافت کنید؛ سپس تفکیک را روشن کنید.':'Iran bypass is not ready. Download valid '+missing.join(' and ')+' files in Routing data before enabling it.'):'';
 },
 renderGeo:function(){
  if(!this.geoNode)return;var self=this,g=this.geo||{},fa=this.lang==='fa';this.paintGeoReadiness();var supported=!!(this.status.features&&this.status.features.geoManager);
  function row(title,control){return node('div',{'class':'pol-settings-row'},[typeof title==='string'?node('span',{'class':'pol-label'},title):title,control]);}
  function update(kind){self.geoFeedback=null;self.geoAwaiting=true;self.action(function(){return api.geoUpdate(kind);},'verifying','geo').then(function(r){if(r&&r.geoip){self.geo=r;self.renderGeo();}});}
  var rows=['geoip','geosite'].map(function(kind){var data=g[kind]||{},key=kind==='geoip'?'geoipUrl':'geositeUrl',sources=data.sources||[],value=self.s[key],custom=!sources.some(function(x){return x.url===value;});
   var select=node('select',{'class':'pol-input','dir':'ltr','aria-label':kind+' '+self.t('geoSource'),'change':function(ev){if(ev.target.value==='custom'){self.customGeo=kind;self.renderGeo();}else{self.customGeo=null;self.save(key,ev.target.value).then(function(){self.renderGeo();});}}},sources.map(function(x){return node('option',{'value':x.url},x.name);}).concat([node('option',{'value':'custom'},self.t('customSource'))]));select.value=custom||self.customGeo===kind?'custom':value;
   var controls=[select];if(select.value==='custom')controls.push(self.field(key,'customSource'));
   return row((fa?'آدرس به‌روزرسانی ':'Update source ')+(kind==='geoip'?'GeoIP':'GeoSite'),node('div',{'class':'pol-geo-source'},controls));
  });
  if(supported){rows.push(row(this.label('geoDirectory',fa?'فایل‌ها در مسیر انتخاب‌شده نگهداری می‌شوند. پس از تغییر مسیر، فایل‌ها را دوباره دریافت کنید. مسیر مجاز: /etc/pol/geo یا زیر /mnt و /opt.':'Files stay in the selected directory. Download them again after changing it. Use /etc/pol/geo or a directory under /mnt or /opt.'),this.field('geoDirectory').querySelector('input')));rows.push(row(this.t('geoAuto'),this.select('geoAuto',['disabled','daily','weekly']).querySelector('select')));
   ['geoipUpdate','geositeUpdate'].forEach(function(key){var button=node('button',{'type':'button','role':'switch','class':'pol-inline-slider','aria-checked':String(!!self.s[key]),'aria-label':self.t(key),'click':function(){self.save(key,!self.s[key]).then(function(){self.renderGeo();});}},node('span',{'class':'pol-switch-track','aria-hidden':'true'},node('span',{'class':'pol-switch-thumb'})));rows.push(row(self.t(key),button));});}
  rows.push(row(fa?'نسخه قوانین':'Installed rules',node('div',{'class':'pol-geo-versions'},['geoip','geosite'].map(function(kind){var d=g[kind]||{};return node('div',{'class':'pol-geo-version'},[node('strong',{},kind==='geoip'?'GeoIP':'GeoSite'),node('span',{'class':'pol-geo-size'},d.size?(d.size/1048576).toFixed(2)+' MB':'—'),node('small',{},d.updated?new Date(d.updated*1000).toLocaleString(fa?'fa-IR':'en-GB'):self.t('notDownloaded')),self.button('geoUpdate',function(){update(kind);})]);}).concat([g.free!=null?node('small',{'dir':'ltr'},(g.free/1073741824).toFixed(2)+' GB · '+g.directory):null]))));
  this.geoProgressNode=node('div',{'class':'pol-geo-progress','aria-live':'polite'});rows.push(this.geoProgressNode);dom.content(this.geoNode,this.card('geoTitle',rows));this.paintGeoProgress();this.fitFields();this.paintStatus();
 },

	t:function(key){return words[this.lang][key]||key;},
	variants:function(){if(this.s.mode==='worker')return ['normal','fragment'];if(this.s.mode==='warp')return ['normal','obfuscated'];if(this.s.mode==='masque')return ['normal',this.s.transport==='h3'?'obfuscated':'fragment'];if(this.s.mode==='serverless')return ['fragment'];return ['normal'];},
 importedDeleteButton:function(){var self=this;return node('button',{'type':'button','class':'pol-btn pol-delete-button','data-pol-managed':'import-delete','disabled':!this.s.custom,'click':function(){var update={custom:null};if(self.s.mode==='custom'){update.mode='worker';update.enabled=false;}self.action(function(){return api.save(update);});}},[icon('trash'),node('span',{},this.lang==='fa'?'حذف':'Delete')]);},
 accountStatus:function(name,account){return node('p',{},[name+': ',node('span',{'style':'color:'+(account?'#059669':'#dc2626')+';font-weight:800'},this.t(account?'registered':'missing'))]);},
 accountDeleteButton:function(kind){
  var self=this,name=kind==='warp'?'WARP':'MASQUE';
  var button=node('button',{'type':'button','class':'pol-btn','style':'background:#dc2626!important;color:#fff!important;border-color:#dc2626!important','aria-label':this.lang==='fa'?'حذف اکانت '+name:'Delete '+name+' account','click':function(){
   if(self.s.enabled&&self.s.mode===kind){self.message(self.lang==='fa'?'ابتدا اتصال '+name+' را خاموش کنید.':'Turn off the '+name+' connection first.',true);return;}
   var update={};update[kind]=null;self.action(function(){return api.save(update);});
  }},[icon('trash'),node('span',{},this.lang==='fa'?'حذف':'Delete')]);
  button.disabled=!this.s[kind];return button;
 },

	button:function(key,fn,cls){return node('button',{'class':'pol-btn '+(cls||''),'style':key==='useEndpoint'?'background:#059669!important;color:#fff!important;border-color:#059669!important':key==='scanPause'||key==='paused'?'background:#dc2626!important;color:#fff!important;border-color:#dc2626!important':null,'type':'button','click':fn},[icon(({connection:'settings',main:'power',check:'refresh',updateAction:'updates',download:'updates',updateCore:'updates',updateApp:'updates',registerWarp:'cloudflare',registerMasque:'cloudflare',deploy:'cloudflare'})[key]||key),node('span',{},this.t(key==='connection'?'settings':key))]);},
 scanError:function(text){
  if(this.lang!=='fa')return text;
  var errors={
   'enter between 1 and 12 unique candidates':'بین ۱ تا ۱۲ آدرس متفاوت وارد کنید؛ هر آدرس را در یک خط بنویسید.',
   'Tor Project sent no bridges of this type':'سرویس Tor برای این نوع اتصال پلی برنگرداند.','scanner is not running':'اسکن فعالی وجود ندارد.','scanner supports Worker, WARP and MASQUE':'اسکن فقط برای Worker، WARP و MASQUE قابل استفاده است.',
   'use host:port or [IPv6]:port for tunnel endpoints':'برای WARP و MASQUE، آدرس را همراه پورت وارد کنید؛ مانند 162.159.192.1:2408 یا [IPv6]:2408.',
   'invalid endpoint port':'پورت باید عددی بین ۱ تا ۶۵۵۳۵ باشد.',
   'invalid scanner hostname':'آدرس اسکن معتبر نیست؛ فقط IP یا دامنه را بدون https:// و مسیر وارد کنید.',
   'scanner core did not start':'هستهٔ آزمایشی اسکن اجرا نشد.',
   'HTTPS probe failed or timed out':'آزمایش اتصال HTTPS ناموفق بود یا مهلت آن پایان یافت.',
   'WARP account is missing':'ابتدا اکانت WARP را بسازید.',
   'MASQUE account is missing':'ابتدا اکانت MASQUE را بسازید.',
   'Worker hostname and valid UUID are required':'ابتدا دامنهٔ Worker و UUID معتبر را وارد کنید.',
   'another operation is in progress':'عملیات دیگری در حال اجراست؛ پس از پایان آن دوباره امتحان کنید.',
   'another background operation is in progress':'عملیات دیگری در پس‌زمینه در حال اجراست؛ پس از پایان آن دوباره امتحان کنید.',
   'Operation timed out; check Logs.':'مهلت اسکن پایان یافت؛ گزارش‌ها را بررسی کنید.'
  };
  if(errors[text])return errors[text];
  return String(text).replace(/probe returned HTTP (\d+)/g,'آزمایش اتصال با کد HTTP برابر $1 پاسخ داد.');
 },
 paintScanFeedback:function(){
  if(!this.scanFeedbackNode)return;
  var job=this.status.job||{}, progress=job.progress;
  var active=job.method==='scan'&&(job.state==='pending'||job.state==='running');
  var f=this.scanFeedback;
  if(active&&!(f&&f.error)){
   var paused=progress&&progress.phase==='paused';
   var text=this.lang==='fa'?(paused?'اسکن موقتاً متوقف است':'در حال اسکن'):(paused?'Scan paused':'Scanning');
   if(progress&&progress.total>0)text+=' · \u2066'+(progress.received||0)+' / '+progress.total+'\u2069';
   f={text:text,error:false};
  }
  if(this.scanEmptyNode)this.scanEmptyNode.hidden=Boolean(active||this.actionTarget==='scan'||f);
  this.scanFeedbackNode.textContent=f?f.text:'';
  this.scanFeedbackNode.className='pol-flash'+(f&&f.error?' is-error':'');
  this.scanFeedbackNode.setAttribute('role',f&&f.error?'alert':'status');
 },
 updateError:function(text){return this.lang==='fa'&&/context deadline exceeded|Client.Timeout|context cancellation while reading body/.test(String(text))?'مهلت دریافت فایل پایان یافت؛ دانلود کامل نشده و به‌روزرسانی نصب نشده است. دوباره تلاش کنید.':text;},
	message:function(text,error){text=this.updateError(text);var rpcFault=transientRPC({message:text});if(rpcFault)text=this.lang==='fa'?'ارتباط با سرویس پل موقت قطع شده است؛ بررسی ارتباط به صورت خودکار ادامه دارد.':'Pol service is temporarily unavailable; reconnecting automatically.';if(this.actionTarget==='dns'){this.dnsProbeFeedback={text:error?this.dnsProbeError(text):text,error:!!error};this.paintDNSFeedback();return;}if(this.actionTarget==='scan'){this.scanFeedback={text:this.scanError(text),error:!!error};this.paintScanFeedback();return;}if(this.actionTarget==='geo'){this.geoFeedback={text:text,error:!!error};this.paintGeoProgress();return;}if(this.lang==='fa'){var translations={
 'Cloudflare usage could not be loaded':'آمار کلادفلر دریافت نشد. دسترسی Analytics توکن و اتصال اینترنت را بررسی کنید.',
 'Cloudflare sign-in timed out':'مهلت اتصال به کلادفلر تمام شد. دوباره تلاش کنید.',
 'Cloudflare account access could not be verified':'دسترسی به حساب تأیید نشد. شناسهٔ حساب، اعتبار توکن و مجوز سرویس انتخاب‌شده را بررسی کنید.',
 'invalid Cloudflare product':'سرویس کلادفلر را انتخاب کنید.',
 'SSH host, valid port and username are required':'آدرس سرور، پورت معتبر و نام کاربری SSH را وارد کنید.',
 'SSH host fingerprint is required':'فینگرپرینت کلید میزبان SSH را وارد کنید.',
 'SSH host fingerprint must be in SHA256 format':'فینگرپرینت کلید میزبان SSH باید در قالب SHA256 باشد.',
 'SSH password or private key is required':'رمز عبور یا کلید خصوصی SSH را وارد کنید.',
 'invalid SSH port or keepalive interval':'پورت یا بازهٔ Keepalive برای SSH معتبر نیست.',
 'invalid SSH server, username or private key':'آدرس سرور، نام کاربری یا کلید خصوصی SSH معتبر نیست.',
 'SSH chains require a TCP-capable proxy':'زنجیرهٔ SSH به پروکسی سازگار با TCP نیاز دارد.',
 'the private key is encrypted: set its passphrase':'رمز کلید خصوصی را وارد کنید.',
 'wrong passphrase for the private key':'رمز کلید خصوصی درست نیست.',
 'choose only one final exit':'برای خروجی نهایی، فقط یکی از زنجیرهٔ پروکسی یا Tor/سایفون را انتخاب کنید.','turn off the Worker connection first':'ابتدا اتصال ورکر را خاموش کنید.','select a saved deployment before deleting it':'ابتدا یک استقرار ذخیره‌شده را انتخاب کنید.','add at least one LAN device address':'ابتدا IP حداقل یک دستگاه را به فهرست اضافه کنید.','invalid LAN device address':'آدرس دستگاه معتبر نیست؛ IP یا CIDR وارد کنید.','add custom Tor bridges first':'ابتدا پل‌های سفارشی Tor را وارد کنید.','probe interval must be 10-3600 seconds':'بازه بررسی باید بین ۱۰ و ۳۶۰۰ ثانیه باشد.','invalid exit country':'کد کشور خروجی معتبر نیست.','invalid UDP noise':'تنظیمات نویز UDP معتبر نیست.','too many UDP noise packets':'تعداد بسته‌های نویز UDP بیش از حد مجاز است.','context deadline exceeded':'مهلت دریافت فایل پایان یافت؛ دانلود کامل نشده و به‌روزرسانی نصب نشده است. دوباره تلاش کنید.','Client.Timeout':'مهلت دریافت فایل پایان یافت؛ دانلود کامل نشده و به‌روزرسانی نصب نشده است. دوباره تلاش کنید.','Worker hostname and valid UUID are required':'دامنه ورکر و شناسه UUID معتبر را وارد کنید.','invalid API token':'توکن API معتبر نیست؛ مقدار کامل توکن ساخته شده در Cloudflare را وارد کنید.','invalid Account ID':'شناسه حساب معتبر نیست؛ مقدار ۳۲ کاراکتری Account ID را وارد کنید.','Worker name already exists; choose a new name':'این نام ورکر از قبل وجود دارد؛ یک نام جدید انتخاب کنید.','choose a workers.dev subdomain in Cloudflare Workers & Pages first':'ابتدا در Workers & Pages کلادفلر یک زیر دامنه workers.dev تعیین کنید.','invalid Worker hostname':'دامنه ورکر معتبر نیست؛ آن را بدون https:// و بدون مسیر وارد کنید.'};Object.keys(translations).some(function(key){if(String(text).indexOf(key)>=0){text=translations[key];return true;}return false;});}if(this.flash){this.flash.dataset.rpcFault=String(rpcFault);this.flash.textContent=text;this.flash.className='pol-flash'+(error?' is-error':''); this.flash.setAttribute('role',error?'alert':'status');}},
	request:function(p){return p.then(function(r){if(r&&r.error)throw new Error(r.error);return r;});},
	waitJob:function(result){
		var self=this;if(this.actionTarget==='geo')this.geoAwaiting=false;if(!result.job)return Promise.resolve(result);var deadline=Date.now()+600000;
		return new Promise(function(resolve,reject){function check(){api.status().then(function(s){self.status=s;self.paintStatus();self.paintUpdateProgress();self.paintGeoProgress();self.paintScanFeedback();var job=s.job||{};if(job.id===result.job&&job.state==='done')return resolve(job.result||{});if(job.id===result.job&&job.state==='failed')return reject(new Error(job.error));if(Date.now()>deadline)return reject(new Error('Operation timed out; check Logs.'));setTimeout(check,1500);}).catch(function(error){if(transientRPC(error)&&Date.now()<deadline){setTimeout(check,1500);return;}reject(error);});}check();});
	},
	action:function(fn,label,target){
		var self=this;if(this.busy)return Promise.resolve();this.actionTarget=target||null;if(target==='scan')this.scanFeedback=null;this.busy=true;this.paintStatus();this.message(this.t(label||'starting'));
		return this.queue.then(fn).then(function(r){return self.request(Promise.resolve(r));}).then(function(r){return self.waitJob(r);}).then(function(r){return Promise.all([api.settings(),api.status()]).then(function(v){self.s=v[0];self.status=v[1];self.message(r.scheduled?self.t('scheduled'):self.t('applied'));self.panel();return r;});}).catch(function(e){self.message(e.message,true);}).finally(function(){self.busy=false;self.actionTarget=null;self.geoAwaiting=false;self.paintGeoProgress();self.paintStatus();if(self.tab==='updates')self.renderReleases();});
	},
	save:function(key,value){
		var self=this;this.s[key]=value;this.message(this.t('applying'));
		this.queue=this.queue.catch(function(){}).then(function(){return self.request(api.save(Object.fromEntries([[key,value]])));}).then(function(){self.message(self.t('applied'));return Promise.all([api.settings(),api.status()]);}).then(function(v){self.s=v[0];self.status=v[1];self.paintStatus();if(key==='geoDirectory'){self.geo=null;self.panel();}}).catch(function(e){self.message(e.message,true);return api.settings().then(function(s){self.s=s;self.panel();});});
		return this.queue;
	},
	field:function(key,label,type,help){
		var self=this, opts={'class':'pol-input','dir':'ltr','type':type||'text','value':this.s[key]==null?'':this.s[key],'change':function(ev){self.save(key,type==='number'?Number(ev.target.value):ev.target.value);}};
		if(type==='number'){opts.min=key==='warpKeepAlive'?0:1;opts.max=65535;if(key==='warpMtu'){opts.min=1280;opts.max=1500;}}
		return node('label',{'class':'pol-field'},[this.label(label||key,this.helpText(key)||(help?this.t(help):null)),node('input',opts)]);
	},
	select:function(key,values,label){
		var self=this, select=node('select',{'class':'pol-input','change':function(ev){
   var value=ev.target.value;
   if(key==='mode'){
    if(value==='ssh'&&self.s.mode!=='ssh'&&self.status.enabled){
     self.sshDraft=Object.assign({},self.s.ssh);
     self.panel();
     return;
    }
    if(self.sshDraft){self.sshDraft=null;if(value===self.s.mode){self.panel();return;}}
   }
   self.save(key,value).then(function(){self.panel();});
  }},values.map(function(v){return node('option',{'value':v},key==='torBridgeMode'&&['auto','direct','custom'].indexOf(v)>=0?(self.lang==='fa'?({auto:'خودکار',direct:'بدون پل',custom:'پل سفارشی'})[v]:v):key==='exitMode'&&v==='none'?(self.lang==='fa'?'بدون خروجی دوم':'No second exit'):v===''?(self.lang==='fa'?'خودکار':'Automatic'):self.t(v));}));select.value=(key==='mode'&&this.sshDraft?'ssh':this.s[key])||(key==='exitMode'?'none':'');return node('label',{'class':'pol-field'},[this.label(label||key,this.helpText(key)),select]);
	},
	switchField:function(key){var self=this;if(key==='routeIran'||key==='transparent'||key==='bestPing'||key==='fakeSniDns'||key==='fakeDns'||key==='tcpFastOpen'||key==='warpReserved'||key==='lanSharing'){var slider=node('button',{'type':'button','role':'switch','class':'pol-inline-slider','aria-checked':String(!!this.s[key]),'aria-label':this.t(key),'click':function(){slider.disabled=true;self.save(key,!self.s[key]).then(function(){slider.setAttribute('aria-checked',String(!!self.s[key]));slider.disabled=false;});}},node('span',{'class':'pol-switch-track','aria-hidden':'true'},node('span',{'class':'pol-switch-thumb'})));return node('div',{'class':'pol-inline-control'},[this.label(key),slider]);}var input=node('input',{'type':'checkbox','change':function(ev){self.save(key,ev.target.checked);}});input.checked=!!this.s[key];return node('label',{'class':'pol-switch'+(key==='tcpFastOpen'?' pol-switch-ltr':'')},[input,this.label(key)]);},
	card:function(title,children){var details=this.helpText(title)||'';children=children.filter(function(child){if(child&&child.tagName==='P'&&!child.classList.contains('pol-note')){details+=(details?'\n\n':'')+child.textContent;return false;}return true;});return node('section',{'class':'pol-card'+(title==='block'?' pol-block-card':'')},[node('h3',{},this.label(title,details))].concat(children));},
	draw:function(){
		var self=this;this.root.className='pol '+((this.theme==='dark'||this.theme==='auto'&&this.media.matches)?'pol-dark':'');this.root.dir=this.lang==='fa'?'rtl':'ltr';this.root.lang=this.lang;
		this.flash=node('div',{'class':'pol-flash','aria-live':'polite'});
		this.mainButton=this.button('main',function(){if(self.status.enabled){self.busy=true;self.paintStatus();self.request(api.toggle(false)).then(function(){return Promise.all([api.settings(),api.status()]);}).then(function(v){self.s=v[0];self.status=v[1];self.message(self.t('applied'));}).catch(function(e){self.message(e.message,true);}).finally(function(){self.busy=false;self.paintStatus();});}else{self.action(function(){
   if(self.s.mode==='ssh')self.checkSSHInputs(self.s.ssh);
   return api.toggle(true);
  });}},'pol-power');
		this.statusNode=node('span',{'class':'pol-status'});this.content=node('div',{'class':'pol-content'});
		var language=node('select',{'class':'pol-language','aria-label':'Language','change':function(ev){self.lang=ev.target.value;localStorage.setItem('pol-language',self.lang);self.draw();}},[node('option',{'value':'fa'},'فارسی'),node('option',{'value':'en'},'English')]);language.value=this.lang;
		this.uptimeNode=node('span',{'class':'pol-uptime-value','dir':'auto'});
		var uptime=node('div',{'class':'pol-uptime'},[node('span',{},this.t('uptime')),this.uptimeNode]);
		var theme=node('div',{'class':'pol-theme-controls','role':'group','aria-label':this.t('auto')},['auto','light','dark'].map(function(key){var b=self.button(key,function(){self.theme=key;localStorage.setItem('pol-theme',key);self.draw();},'glass'+(self.theme===key?' theme-selected':''));b.setAttribute('aria-pressed',String(self.theme===key));return b;}));
		dom.content(this.root,[node('header',{'class':'pol-hero'},[node('div',{'class':'pol-brand'},[node('img',{'class':'pol-logo','src':L.resource('pol/logo.svg')+'?v=bridge-6','alt':'','width':90,'height':64}),node('div',{},[node('h2',{},[this.t('title'),node('span',{'class':'pol-version'},'v'+(this.status.version||''))]),node('p',{},this.t('subtitle'))])]),node('div',{'class':'pol-tools'},[theme,node('div',{'class':'pol-language-column'},[language,uptime])])]),node('nav',{'class':'pol-nav','aria-label':this.t('title')},['home','connection','cloudflare','routing','updates','reports'].map(function(key){return self.button(key,function(){self.tab=key;if(key==='routing')self.geo=null;self.draw();},'pol-tab-'+key+(self.tab===key?' selected':''));})),this.flash,this.content,node('footer',{'class':'pol-footer'},[node('img',{'src':L.resource('pol/logo.svg')+'?v=bridge-6','alt':'Pol','width':64,'height':44}),node('div',{},'Pol · Copyright © 2026 dreamboxone · GPLv3 / AGPLv3 · '+(this.lang==='fa'?'بدون ضمانت':'No warranty')),node('div',{'class':'pol-footer-links'},[['support','https://t.me/routekernel1','telegram'],['source','https://github.com/dreamboxone/pol','source']].map(function(v){return node('a',{'href':v[1],'target':'_blank','rel':'noopener noreferrer'},[icon(v[2]),node('span',{},v[0]==='support'?'Telegram · t.me/routekernel1':self.t(v[0]))]);}))])]);
		this.panel();this.paintStatus();
		this.root.querySelectorAll('.pol-theme-controls button').forEach(function(b,i){b.title=self.t(['auto','light','dark'][i]);b.setAttribute('aria-label',b.title);});
	},
	paintStatus:function(){
		if(!this.mainButton)return;
		if(this.uptimeNode){var seconds=Math.max(0,Math.floor(Number(this.status.uptime)||0)),days=Math.floor(seconds/86400),time=[Math.floor(seconds%86400/3600),Math.floor(seconds%3600/60),seconds%60].map(function(v){return String(v).padStart(2,'0');}).join(':');this.uptimeNode.textContent=(days?days+' '+this.t('days')+' · ':'')+time;}
		if(this.content)this.content.querySelectorAll('input,select,textarea').forEach(function(el){el.disabled=this.busy;},this);
		if(this.content)this.content.querySelectorAll('button').forEach(function(el){if(el.dataset.polManaged==='site')el.disabled=this.busy||!this.status.running||!(this.status.features&&this.status.features.siteChecks);else if(el.dataset.polManaged==='scan-control'){var j=this.status.job||{};el.disabled=j.method!=='scan'||(j.state!=='pending'&&j.state!=='running');}else if(el.dataset.polManaged==='import-delete')el.disabled=this.busy||!this.s.custom;else if(el.dataset.polManaged!=='release')el.disabled=this.busy;},this);
		this.mainButton.disabled=this.busy&&!this.status.enabled;this.mainButton.className='pol-btn pol-power '+(this.status.enabled?'is-on':'')+(this.busy?' pol-main-busy':'');dom.content(this.mainButton,[node('span',{'class':'pol-switch-track','aria-hidden':'true'},node('span',{'class':'pol-switch-thumb'})),node('span',{},this.t(this.status.enabled?'on':'off'))]);this.mainButton.setAttribute('role','switch');this.mainButton.setAttribute('aria-checked',String(!!this.status.enabled));this.mainButton.setAttribute('aria-busy',String(this.busy));this.mainButton.setAttribute('aria-checked',String(!!this.status.enabled));
		this.statusNode.textContent=this.t(this.status.running?'connected':'disconnected');this.statusNode.className='pol-status '+(this.status.running?'is-on':'');
		if(this.tab==='routing')this.paintGeoReadiness();
		if(this.stats&&!(this.activeHelp&&this.stats.contains(this.activeHelp)))dom.content(this.stats,this.dashboardCards());
	},
	statCards:function(){var self=this;return [['core',((this.status.coreVersion||'').match(/^Xray\s+([^\s]+)/)||[])[1]||'—'],['firewall',this.t(this.status.firewall?'active':'inactive')],['package',(this.status.packageManager||'').toUpperCase()]].map(function(v){return node('div',{'class':'pol-stat'},[node('small',{},self.t(v[0])),node('b',{'dir':'auto'},v[1]||'—')]);});},
 cloudflareGuide:function(){
  var fa=this.lang==='fa';
  function link(label,url){return node('a',{'href':url,'target':'_blank','rel':'noopener noreferrer'},label);}
  function step(title,body){return node('li',{},[node('strong',{},title),node('div',{},body)]);}
  var steps=fa?[
   step('وارد اکانت Cloudflare شوید',[link('باز کردن پنل Cloudflare','https://dash.cloudflare.com/'),'؛ با حساب فعلی وارد شوید و حساب مورد نظر را انتخاب کنید.']),
   step('زیر دامنه workers.dev را آماده کنید',[link('باز کردن Workers & Pages','https://dash.cloudflare.com/?to=/:account/workers-and-pages'),'؛ در بخش Your subdomain یک نام برای حساب تعیین کنید. اگر از قبل نشانی مانند myname.workers.dev دارید، همین مرحله آماده است. خرید دامنه لازم نیست.']),
   step('اکانت آی‌دی را کپی کنید','در پنل کلادفلر Ctrl+K را بزنید، Copy account ID را جستجو و انتخاب کنید. راه دیگر: Workers & Pages، بخش Account Details و دکمه کپی کنار Account ID. این مقدار ۳۲ کاراکتر دارد؛ آن را در کادر شناسه حساب پایین وارد کنید، نه Zone ID.'),
   step('در صورت نیاز توکن بسازید',[link('باز کردن API Tokens','https://dash.cloudflare.com/profile/api-tokens'),'؛ Create Token را بزنید و در Token name نامی مانند Pol Worker وارد کنید. در بخش Policies روی Start from scratch (Custom) بزنید و Start with an Account policy را انتخاب کنید. اگر کادر Edit Account policy را می‌بینید، این مرحله انجام شده است؛ در همان کادر ادامه دهید و Add new policy را نزنید.']),
   step('دسترسی توکن را تعیین کنید','در کادر Edit Account policy، گزینه Include را نگه دارید و به جای All accounts، اکانت مورد نظر خود را انتخاب کنید. سپس Developer Platform را باز کنید؛ و در بخش Workers Scripts دسترسی Edit آن را انتخاب کنید.'),
   step('توکن را بسازید و کپی کنید','Continue to summary و سپس Create Token را بزنید. مقدار کامل توکن فقط یک بار نمایش داده می شود؛ آن را کپی کنید و در کادر توکن API پایین بگذارید. توکن با Account ID فرق دارد؛ Global API Key را در این کادر نگذارید. توکن در تنظیمات پل ذخیره نمی شود.'),
   step('فرم پایین را تکمیل و ورکر را بسازید','توکن API و شناسه حساب همان حساب را وارد کنید. برای نام ورکر، یک نام جدید مانند pol-home-123 انتخاب کنید؛ نام باید با حرف کوچک انگلیسی شروع شود و ۳ تا ۴۱ کاراکتر از حروف کوچک انگلیسی، عدد یا خط تیره داشته باشد. سپس ساخت Worker را بزنید. برای این مرحله روتر باید به اینترنت و API کلادفلر دسترسی داشته باشد.'),
   step('اتصال را روشن کنید','پس از پیام موفقیت، دامنه ورکر و UUID به صورت خودکار در خانه، بخش تنظیمات پایه قرار می گیرند. روش معمولی را انتخاب و دکمه اصلی را روشن کنید؛ کارت وضعیت باید اینترنت در دسترس است را نشان دهد.')
  ]:[
   step('Sign in to Cloudflare',[link('Open Cloudflare dashboard','https://dash.cloudflare.com/'),'. Sign in and select your existing account.']),
   step('Set up workers.dev',[link('Open Workers & Pages','https://dash.cloudflare.com/?to=/:account/workers-and-pages'),'. Set Your subdomain if you do not already have an address such as myname.workers.dev. No purchased domain is required.']),
   step('Copy the Account ID','Press Ctrl+K in Cloudflare, search for Copy account ID and select it. Alternatively, copy Account ID under Workers & Pages > Account Details. Paste the 32-character value into Account ID below, not Zone ID.'),
   step('Create an API token if needed',[link('Open API Tokens','https://dash.cloudflare.com/profile/api-tokens'),'. Select Create Token and enter a Token name such as Pol Worker. Under Policies, select Start from scratch (Custom), then Start with an Account policy. Zone policy and User policy are not appropriate for this Worker deployment.']),
   step('Choose permissions and account','In the new permission model, grant Admin at the Workers product scope for this account; Editor cannot create a new Worker. In the legacy Permissions form, choose Account / Workers Scripts / Edit and Account Resources: Include / Specific account for the same account. This workers.dev deployment does not require DNS or Zone permissions. Any IP restriction must allow the router public egress IP, not its LAN address.'),
   step('Create and copy the token','Select Continue to summary, then Create Token. Copy the full secret immediately; Cloudflare only displays it once. Paste it in API token below. Do not paste the Account ID or Global API Key there. Pol does not store the token in settings.'),
   step('Complete the form and deploy','Enter the token and Account ID for the same account. Choose a new Worker name such as pol-home-123: 3–41 lowercase letters, digits or hyphens, starting with a letter. Select Deploy Worker. The router needs access to the internet and Cloudflare API.'),
   step('Connect','After deployment, Pol fills the Worker hostname and UUID under Home > Basic settings. Select Normal and turn on Main Switch. Confirm Internet reachable in the status card.')
  ];
  return node('div',{'class':'pol-cf-guide'},[node('h4',{},fa?'راهنمای مرحله به مرحله':'Step-by-step setup'),node('ol',{},steps),node('div',{'class':'pol-cf-references'},[link(fa?'راهنمای رسمی ساخت توکن':'Official token guide','https://developers.cloudflare.com/fundamentals/api/get-started/create-token/'),link(fa?'راهنمای اکانت آی‌دی':'Account ID guide','https://developers.cloudflare.com/fundamentals/account/find-account-and-zone-ids/'),link(fa?'راهنمای دسترسی Workers':'Workers permissions','https://developers.cloudflare.com/workers/authorization/workers/'),link('workers.dev','https://developers.cloudflare.com/workers/configuration/routing/workers-dev/')])]);
 },
 checkSSHInputs:function(config){
  config=config||{};
  var problem=!config.host||!config.user||!Number.isInteger(config.port)||config.port<1||config.port>65535?
   'SSH host, valid port and username are required':
   !config.hostKey?'SSH host fingerprint is required':
   !/^SHA256:[A-Za-z0-9+/]{43}$/.test(config.hostKey)?'SSH host fingerprint must be in SHA256 format':
   !config.password&&!config.privateKey?'SSH password or private key is required':'';
  if(problem)throw new Error(problem);
 },
 sshSettings:function(){
  var self=this, draft=this.sshDraft, config=draft||this.s.ssh||{};
  var help=this.lang==='fa'?{
   sshHost:'نام دامنه یا IP سرور SSH را وارد کنید.',
   port:'پورت SSH سرور. مقدار پیش‌فرض ۲۲ است.',
   sshUser:'نام کاربری حساب SSH روی سرور.',
   sshHostKey:'فینگرپرینت کلید میزبان با پیشوند SHA256: است و باید از منبعی مورد اعتماد گرفته شود.',
   sshPassword:'برای ورود با رمز عبور، رمز حساب SSH را وارد کنید.',
   sshPrivateKey:'محتوای کامل کلید خصوصی OpenSSH یا PEM را وارد کنید.',
   sshPassphrase:'فقط برای کلید خصوصی رمزگذاری‌شده لازم است.',
   sshKeepAlive:'فاصلهٔ پیام‌های بررسی اتصال SSH به ثانیه. صفر آن را غیرفعال می‌کند.',
   sshTCPKeepAlive:'فاصلهٔ بررسی اتصال TCP به ثانیه. صفر آن را غیرفعال می‌کند.'
  }:{
   sshHost:'Enter the SSH server hostname or IP address.',
   port:'SSH port on the server. The default is 22.',
   sshUser:'Username of the SSH account on the server.',
   sshHostKey:'Use the SHA256 host key fingerprint obtained from a trusted source.',
   sshPassword:'Enter the SSH account password for password authentication.',
   sshPrivateKey:'Enter the complete OpenSSH or PEM private key.',
   sshPassphrase:'Required for encrypted private keys.',
   sshKeepAlive:'Seconds between SSH keepalive checks. Zero disables them.',
   sshTCPKeepAlive:'Seconds between TCP keepalive checks. Zero disables them.'
  };
  var fields=[['host','sshHost','text'],['port','port','number'],['user','sshUser','text'],
   ['hostKey','sshHostKey','text'],['password','sshPassword','password'],
   ['privateKey','sshPrivateKey','textarea'],['passphrase','sshPassphrase','password'],
   ['keepAliveInterval','sshKeepAlive','number'],['tcpKeepAliveInterval','sshTCPKeepAlive','number']];
  return node('fieldset',{'class':'pol-ssh-settings'},[
   node('legend',{},this.t('ssh')),
   node('small',{},this.t('sshHelp')),
   node('div',{'class':'pol-grid'},fields.map(function(field){
    var key=field[0], type=field[2];
    var attrs={'class':'pol-input','dir':'ltr','autocomplete':'off','change':function(ev){
     var value=type==='number'?Number(ev.target.value):ev.target.value;
     if(key==='host'||key==='user'||key==='hostKey')value=value.trim();
     if(draft){draft[key]=value;return;}
     self.save('ssh',Object.assign({},self.s.ssh,{[key]:value}));
    }};
    if(type!=='textarea'){attrs.type=type;attrs.value=config[key]==null?'':config[key];}
    if(type==='number'){attrs.min=key==='port'?1:0;attrs.max=key==='port'?65535:3600;}
    return node('label',{'class':'pol-field'},[self.label(field[1],help[field[1]]),node(type==='textarea'?'textarea':'input',attrs,type==='textarea'?(config[key]||''):null)]);
   })),
   draft?node('div',{'class':'pol-log-tools'},[
    node('button',{'type':'button','class':'pol-btn','click':function(){
     self.action(function(){
      self.checkSSHInputs(draft);
      return api.save({mode:'ssh',ssh:draft});
     }).then(function(result){if(result){self.sshDraft=null;self.panel();}});
    }},this.lang==='fa'?'اعمال و تغییر اتصال به SSH':'Apply and switch to SSH'),
    node('button',{'type':'button','class':'pol-btn','click':function(){self.sshDraft=null;self.panel();}},this.lang==='fa'?'انصراف':'Cancel')
   ]):null
  ]);
 },
 savedChainsCard:function(){
  var self=this,chains=this.s.chains||[];
  var title=node('input',{'class':'pol-input','type':'text','required':true,'maxlength':128});
  var link=node('textarea',{'class':'pol-input pol-code','dir':'ltr','rows':3,'required':true});
  var editing='';
  var rows=chains.map(function(chain){
   return node('div',{'class':'pol-update-row'},[node('strong',{},chain.name),node('div',{'class':'pol-log-tools'},[
    node('button',{'class':'pol-btn','type':'button','click':function(){editing=chain.id;title.value=chain.name;link.value=chain.link;title.focus();}},self.lang==='fa'?'ویرایش':'Edit'),
    self.button('remove',function(){
     var assignments=Object.assign({},self.s.exitChains);
     Object.keys(assignments).forEach(function(mode){if(assignments[mode]===chain.id)delete assignments[mode];});
     self.action(function(){return api.save({chains:chains.filter(function(item){return item.id!==chain.id;}),exitChains:assignments});});
    })
   ])]);
  });
  var form=node('form',{'submit':function(ev){
   ev.preventDefault();
   var name=title.value.trim(),raw=link.value.trim();
   if(!name||!raw)return;
   var item={id:editing||Array.from(crypto.getRandomValues(new Uint32Array(4))).join('-'),name:name,link:raw};
   var next=editing?chains.map(function(chain){return chain.id===editing?item:chain;}):chains.concat(item);
   self.action(function(){return api.save({chains:next});});
  }},[
   node('div',{'class':'pol-grid pol-option-grid'},[
    node('label',{'class':'pol-field'},[this.label('nodeName'),title]),
    node('label',{'class':'pol-field'},[node('span',{'class':'pol-label'},this.lang==='fa'?'لینک یا کانفیگ WireGuard':'URI or WireGuard configuration'),link])
   ]),
   node('button',{'class':'pol-btn','type':'submit'},this.lang==='fa'?'ذخیرهٔ زنجیره':'Save chain')
  ]);
  return this.card('savedChains',rows.concat([form]));
 },
 chainSelect:function(){
  var self=this, selected=(this.s.exitChains||{})[this.s.mode];
  var choices=[node('option',{'value':''},this.lang==='fa'?'بدون زنجیره':'No chain')];
  if(this.s.chain)choices.unshift(node('option',{'value':'@legacy'},this.lang==='fa'?'زنجیرهٔ عمومی':'Global chain'));
  (this.s.chains||[]).forEach(function(chain){choices.push(node('option',{'value':chain.id},chain.name));});
  var select=node('select',{'class':'pol-input','change':function(ev){
   var next=Object.assign({},self.s.exitChains);
   if(ev.target.value==='@legacy')delete next[self.s.mode];else next[self.s.mode]=ev.target.value;
   var update={exitChains:next};
   if(ev.target.value)update.exitMode='none';
   self.action(function(){return api.save(update);});
  }},choices);
  select.value=selected==null?(this.s.chain?'@legacy':''):selected;
  return node('label',{'class':'pol-field'},[this.label('selectedChain'),select]);
 },
 scanOptionFields:function(){
  var self=this,values=this.scanOptions||(this.scanOptions={});
  var fields=[
   ['probeCount',100,1,1000,'تعداد آدرس‌ها','Candidate count'],['outputCount',10,1,1000,'تعداد نتیجه‌ها','Result count'],
   ['attemptCount',3,1,5,'تلاش برای هر آدرس','Attempts per address'],['attemptInterval',50,10,100,'فاصله تلاش‌ها (میلی‌ثانیه)','Attempt interval (ms)'],
   ['probeInterval',100,100,1000,'فاصله آزمایش آدرس‌ها (میلی‌ثانیه)','Probe interval (ms)'],['port',0,0,65535,'پورت (۰ یعنی پورت اتصال)','Port (0 uses connection port)']
  ].map(function(item){
   return node('label',{'class':'pol-field'},[node('span',{'class':'pol-label'},item[self.lang==='fa'?4:5]),node('input',{'class':'pol-input','type':'number','value':values[item[0]]==null?item[1]:values[item[0]],'min':item[2],'max':item[3],'input':function(ev){values[item[0]]=Number(ev.target.value);}})]);
  });
  fields.push(node('label',{'class':'pol-field'},[node('span',{'class':'pol-label'},this.lang==='fa'?'بازه‌های CIDR سفارشی':'Custom CIDR ranges'),node('textarea',{'class':'pol-input','dir':'ltr','rows':3,'input':function(ev){values.cidrs=ev.target.value.split(/[\s,]+/).filter(Boolean);}},(values.cidrs||[]).join('\n'))]));
  return node('details',{},[node('summary',{},this.lang==='fa'?'تنظیمات اسکن خودکار':'Automatic scan settings'),node('div',{'class':'pol-grid pol-option-grid'},fields)]);
 },
 basicSettingsCard:function(){
  var self=this,fields=[this.select('mode',['worker','warp','masque','serverless','tor','psiphon','ssh','custom']),this.s.mode==='serverless'?null:this.select('variant',this.variants())];
  if(this.sshDraft){
   return this.card('connection',[node('div',{'class':'pol-grid'},[fields[0],this.sshSettings()])]);
  }
  if(this.s.mode==='worker')fields=fields.concat([this.field('host'),this.field('workerDomain',null,'text','optional'),this.field('uuid',null,'password'),this.field('address',null,'text','optional'),this.field('port',null,'number'),this.field('path')]);
  if(this.s.mode==='warp'||this.s.mode==='masque'){
   fields.push(this.field('endpoint'));
   if(this.s.mode==='masque')fields.push(this.select('transport',['h2','h3']),this.field('sni'));
  }
  if(this.s.mode!=='tor'&&this.s.mode!=='psiphon'&&this.s.mode!=='custom')fields.push(this.select('exitMode',['none','tor','psiphon']));
  if(this.s.mode!=='custom'&&(this.s.mode==='tor'||this.s.mode==='psiphon'||this.s.exitMode==='tor'||this.s.exitMode==='psiphon'))fields.push(this.select('exitRegion',['','US','DE','NL','GB','FR','CA','CH','SE','JP','SG']));
  if(this.s.mode!=='custom'&&(this.s.mode==='tor'||this.s.exitMode==='tor'))fields.push(this.select('torBridgeMode',['auto','direct','obfs4','snowflake-amp','snowflake','webtunnel','meek_lite','custom']),this.listField('torBridges'),node('div',{'style':'display:flex;flex-wrap:wrap;gap:8px'},['obfs4','snowflake','webtunnel','meek_lite'].map(function(kind){return node('button',{'type':'button','class':'pol-btn','click':function(){self.action(function(){return api.torBridges(kind);}).then(function(r){if(r&&r.bridges)self.save('torBridges',r.bridges).then(function(){self.panel();});});}},(self.lang==='fa'?'دریافت پل ':'Get bridges: ')+kind);})));
  if(this.s.mode==='serverless'){
   fields.push(this.select('serverlessMode',['packet','dns','domain-fronting']));
   if(this.s.serverlessMode==='dns')fields.push(this.select('serverlessDnsType',['doh','doq','tcp','udp']),this.listField('serverlessDnsServers'));
  }
  if(this.s.mode==='ssh')fields.push(this.sshSettings());
  if(this.s.mode!=='custom')fields.push(this.field('dns'),this.field('localDns'),this.chainSelect());
  fields.push(this.field('lan'));
  if(['worker','warp','masque'].indexOf(this.s.mode)>=0)fields.push(this.switchField('bestPing'),this.listField(this.s.mode==='worker'?'cleanIps':'endpoints'),this.field('probeInterval',null,'number'));
  fields.filter(Boolean).forEach(function(field){if(field.classList.contains('pol-settings-row'))field.classList.add('pol-compact-list');if(field.classList.contains('pol-inline-control'))field.classList.add('pol-compact-switch');});
  var children=[node('div',{'class':'pol-grid'},fields),this.switchField('transparent')];
  if(this.tab==='home'&&this.s.mode==='custom')children.push(this.button('importTitle',function(){self.tab='connection';self.draw();}));
  return this.card('connection',children);
 },

	panel:function(){
		var self=this,body=[];this.stats=null;this.activeHelp=null;
		if(this.tab==='home'){
			this.stats=node('div',{'class':'pol-dashboard pol-grid-frame'},this.dashboardCards());
			body=[this.stats,node('section',{'class':'pol-main-control'},[this.label('main'),this.mainButton]),this.basicSettingsCard()];
		}else if(this.tab==='connection'){
			var importText=node('textarea',{'class':'pol-input pol-code','dir':'ltr','rows':7,'placeholder':'vless://00000000-0000-4000-8000-000000000000@your-worker.workers.dev:443?encryption=none&security=tls&type=ws&host=your-worker.workers.dev&path=%2F%3Fed%3D2560#Example','aria-label':this.t('importTitle')});
			body=[this.card('advanced',[node('p',{},this.t('advancedHelp')),node('div',{'class':'pol-grid pol-option-grid'},[this.field('workerSni',null,'text','optional'),this.field('fragmentLength','length'),this.field('fragmentDelay','delay'),this.field('chain',null,'password','optional'),this.select('logLevel',['debug','info','warning','error','none']),this.select('tlsFingerprint',['chrome','firefox','safari','ios','android','edge','randomized'],'fingerprint'),this.select('dnsStrategy',['UseIPv4','UseIPv6','UseIP']),this.field('echDns',null,'text','echHelp'),this.dnsProbePanel(),this.listField('dnsServers'),this.jsonField('dnsHosts',true),this.jsonField('customCdn',true),this.field('upstreamRelay'),this.switchField('fakeDns'),this.switchField('fakeSniDns'),this.field('noiseSize'),this.field('noiseDelay'),this.field('noiseReset'),this.field('warpMtu',null,'number'),this.field('warpKeepAlive',null,'number'),this.switchField('tcpFastOpen'),this.switchField('warpReserved')])]),this.card('account',[node('div',{'class':'pol-account-row'},[this.accountStatus('WARP',this.s.warp),node('div',{'style':'display:flex;flex-wrap:wrap;gap:8px'},[this.button('registerWarp',function(){self.action(function(){return api.register('warp');});}),this.accountDeleteButton('warp')])]),node('div',{'class':'pol-account-row'},[this.accountStatus('MASQUE',this.s.masque),node('div',{'style':'display:flex;flex-wrap:wrap;gap:8px'},[this.button('registerMasque',function(){self.action(function(){return api.register('masque');});}),this.accountDeleteButton('masque')])]),node('small',{},this.t('registrationHelp')),node('small',{},this.lang==='fa'?'آماده یعنی اکانت ثبت شده است؛ اتصال باید جداگانه آزمایش شود. حذف اکانت، اطلاعات آن را فقط از پل پاک می‌کند.':'Ready means the account is registered; connectivity requires a separate test. Deleting an account removes its information from Pol only.')]),this.card('importTitle',[node('p',{},this.t('importHelp')),importText,node('small',{},this.t('configNotice')),node('div',{'class':'pol-import-actions'},[this.button('import',function(){self.action(function(){return api.import(importText.value);});}),this.importedDeleteButton()])])];
			var restoreFile=node('input',{'type':'file','accept':'.json,application/json','style':'display:none','aria-label':this.t('restore'),'change':function(){
    var file=restoreFile.files&&restoreFile.files[0];restoreFile.value='';if(!file)return;
    file.text().then(function(text){var settings;try{settings=JSON.parse(text);}catch(e){throw new Error(self.lang==='fa'?'فایل پشتیبان JSON معتبر نیست.':'Invalid backup JSON.');}
     if(!settings||Array.isArray(settings)||typeof settings!=='object'||typeof settings.mode!=='string'||settings.outbounds)throw new Error(self.lang==='fa'?'فایل پشتیبان تنظیمات پل را انتخاب کنید.':'Select a Pol settings backup.');
     return self.action(function(){return api.import(JSON.stringify(settings));});
    }).catch(function(e){self.message(e.message,true);});
   }});
   body.push(this.card('maskRules',[node('small',{},this.lang==='fa'?'قوانین با ساختار BPB وارد می‌شوند. فهرست خالی از تنظیمات ساده استفاده می‌کند. هر قانون فرگمنت packets، lengths و delays دارد؛ نویزها از rand، str، hex، base64، array و exp پشتیبانی می‌کنند.':'Enter BPB-format rules. Empty lists retain the basic settings. Fragment rules contain packets, lengths and delays. Noise supports rand, str, hex, base64, array and exp.'),node('div',{'class':'pol-grid pol-option-grid'},['fragments','udpNoises','masqueFragments','masqueNoises','serverlessFragments','serverlessNoises'].map(function(key){return self.jsonField(key);}))]));
   body.push(this.savedChainsCard());
   body.push(this.card('lanSharing',[node('small',{},this.lang==='fa'?'SOCKS روی پورت 24802 و HTTP روی 24803، فقط روی اینترفیس LAN و با حساب پراکسی.':'SOCKS on port 24802 and HTTP on 24803, bound to the LAN interface with proxy credentials.'),node('div',{'class':'pol-grid pol-option-grid'},[this.field('lanUsername'),this.field('lanPassword',null,'password'),this.switchField('lanSharing')])]));
   body.push(this.card('configView',[this.button('configView',function(){self.showConfig();})]));
   body.push(this.card('backupTitle',[node('p',{},this.t('exportHelp')),restoreFile,node('div',{'style':'display:flex;flex-wrap:wrap;gap:12px;align-items:center'},[this.button('export',function(){self.download('pol-settings.json',JSON.stringify(self.s,null,2),'application/json');}),this.button('restore',function(){restoreFile.click();})])]));
			if(['worker','warp','masque'].indexOf(this.s.mode)>=0){
				var candidates=node('textarea',{'class':'pol-input pol-code','dir':'ltr','rows':5,'aria-label':this.t('scan'),'input':function(ev){self.scanText=ev.target.value;}},this.scanText||'');
				this.scanEmptyNode=node('p',{},this.t('scanEmpty'));
				var results=node('div',{'class':'pol-scan-results'},this.scanEmptyNode);
    var selectedResults=new Set();
				if(this.scanResults&&this.scanResults.mode===this.s.mode&&this.scanResults.variant===this.s.variant&&this.scanResults.transport===this.s.transport){this.scanEmptyNode=null;dom.content(results,this.scanResults.results.map(function(r){return node('div',{'class':'pol-update-row'},[node('div',{},[node('b',{'dir':'ltr'},r.address),node('small',{},r.error?self.t('failed')+' · '+self.scanError(r.error):r.latency+(self.lang==='fa'?' م‌ثانیه':' ms'))]),r.error?null:node('div',{'class':'pol-log-tools'},[
      node('input',{'type':'checkbox','aria-label':(self.lang==='fa'?'انتخاب ':'Select ')+r.address,'change':function(ev){if(ev.target.checked)selectedResults.add(r.address);else selectedResults.delete(r.address);}}),
      self.button('useEndpoint',function(){self.save(self.s.mode==='worker'?'address':'endpoint',r.address);})
     ])]);}).concat(this.scanResults.results.some(function(r){return !r.error;})?[node('button',{'class':'pol-btn','type':'button','click':function(){
      if(!selectedResults.size)return;
      var key=self.s.mode==='worker'?'cleanIps':'endpoints';
      var next=Array.from(new Set((self.s[key]||[]).concat(Array.from(selectedResults))));
      if(next.length>24){self.message(self.lang==='fa'?'حداکثر ۲۴ آدرس ذخیره می‌شود.':'At most 24 addresses can be saved.',true);return;}
      self.save(key,next).then(function(){self.panel();});
     }},this.lang==='fa'?'ذخیرهٔ آدرس‌های انتخاب‌شده':'Save selected addresses')]:[]));}
				this.scanFeedbackNode=node('div',{'aria-live':'polite'});this.paintScanFeedback();
				body.push(this.card('scan',[node('p',{},this.t('scanHelp')),this.scanOptionFields(),node('div',{'class':'pol-scan-input-row'},[candidates,this.button('scan',function(){self.action(function(){return api.scan(candidates.value,{});},null,'scan').then(function(r){if(r&&r.results){self.scanResults=r;self.panel();}});}),this.button('scanAuto',function(){self.action(function(){return api.scan('auto',self.scanOptions||{});},null,'scan').then(function(r){if(r&&r.results){self.scanResults=r;self.panel();}});})]),node('div',{'style':'display:flex;flex-wrap:wrap;gap:8px'},[['scanPause','pause'],['scanResume','resume'],['scanStop','stop']].map(function(pair){var b=self.button(pair[0],function(){self.request(api.scanControl(pair[1])).catch(function(e){self.scanFeedback={text:self.scanError(e.message),error:true};self.paintScanFeedback();});});b.dataset.polManaged='scan-control';return b;})),this.scanFeedbackNode,results]));
			}
		}else if(this.tab==='cloudflare'){
			var deploymentType=node('select',{'class':'pol-input'},[node('option',{'value':'worker'},'Workers'),node('option',{'value':'pages'},'Pages')]);
   var token=node('input',{'class':'pol-input','type':'password','dir':'ltr','autocomplete':'off','spellcheck':'false'}),account=node('input',{'class':'pol-input','type':'password','dir':'ltr','autocomplete':'off','id':'pol-cf-account-id'}),name=node('input',{'class':'pol-input','dir':'ltr','value':'pol-'+Date.now().toString(36).slice(-7)});
			function cfField(key,input){
    if(key!=='accountId')return node('label',{'class':'pol-field'},[node('span',{},self.t(key)),input]);
    var showLabel=self.lang==='fa'?'نمایش اکانت آی‌دی':'Show Account ID',hideLabel=self.lang==='fa'?'پنهان کردن اکانت آی‌دی':'Hide Account ID';
    var eye=node('button',{'type':'button','class':'pol-btn','aria-label':showLabel,'title':showLabel,'aria-controls':'pol-cf-account-id','aria-pressed':'false','click':function(){var visible=input.type==='password';input.type=visible?'text':'password';eye.setAttribute('aria-pressed',String(visible));eye.setAttribute('aria-label',visible?hideLabel:showLabel);eye.title=visible?hideLabel:showLabel;}},icon('eye'));
    input.style.minWidth='0';input.style.flex='1';
    return node('div',{'class':'pol-field'},[node('label',{'for':'pol-cf-account-id'},self.t(key)),node('div',{'style':'display:flex;align-items:center;gap:8px'},[input,eye])]);
   }
   var signInSupported=Boolean(this.status.features&&this.status.features.cloudflareSignIn), session=this.cfSession;
   function credentialFor(profile){
    if(!signInSupported)return token.value.trim();
    if(!session||session.kind!==profile.kind||session.account.toLowerCase()!==profile.accountId.toLowerCase())return '';
    return session.token;
   }
   var accountCard=signInSupported?this.card('cfSignIn',session?[
    node('div',{'class':'pol-note','role':'status'},this.t('cfSignedIn')+' '+(session.kind==='pages'?'Pages':'Workers')),
    node('small',{},this.t('autoWorkerHelp')),
    this.button(session.autoUpdate?'autoWorkerOff':'autoWorkerOn',function(){
     var enabled=!session.autoUpdate;
     self.action(function(){return api.cfAutoUpdate(session.token,session.account,enabled,session.kind);}).then(function(result){if(result){session.autoUpdate=result.enabled;self.panel();}});
    }),
    this.button('cfSignOut',function(){self.cfSession=null;self.cfUsageResult=null;self.message('');self.panel();})
   ]:[
    node('small',{},this.t('cfSignInHelp')),
    node('form',{'submit':function(ev){
     ev.preventDefault();
     var secret=token.value.trim(), id=account.value.trim(), kind=deploymentType.value;
     self.action(function(){return api.cfSignIn(secret,id,kind);},'cfSignIn').then(function(result){
      if(!result)return;
      self.cfSession={account:result.account,kind:result.kind,token:secret,autoUpdate:result.autoUpdate};
      token.value='';
      self.message(self.t('cfSignedIn'));
      self.panel();
     });
    }},[
     node('div',{'class':'pol-grid pol-option-grid'},[cfField('deploymentType',deploymentType),cfField('accountId',account),cfField('token',token)]),
     node('button',{'class':'pol-btn','type':'submit'},this.t('cfSignIn'))
    ])
   ]):null;
   function credentialError(){
    self.message(self.t(signInSupported?'cfSignInRequired':'workerTokenRequired'),true);
   }
   var removeWorker=node('button',{'type':'button','class':'pol-btn','style':'background:#dc2626!important;color:#fff!important;border-color:#dc2626!important','click':function(){
    if(self.s.enabled&&self.s.mode==='worker'){self.message(self.lang==='fa'?'ابتدا اتصال ورکر را خاموش کنید.':'Turn off the Worker connection first.',true);return;}
    self.action(function(){return api.save({host:'',uuid:'',address:'',workerSni:'',workerProfiles:(self.s.workerProfiles||[]).filter(function(p){return p.host!==self.s.host;})}).then(function(r){if(!r.error)self.deployedHost=null;return r;});});
   }},[icon('trash'),node('span',{},this.t('deleteWorker'))]);
   var selectedProfile=(this.s.workerProfiles||[]).filter(function(p){return p.host===self.s.host;})[0];
   var refreshWorker=selectedProfile&&['worker','pages'].indexOf(selectedProfile.kind)>=0&&selectedProfile.accountId&&selectedProfile.project?
    this.button('refreshWorker',function(){
     var secret=credentialFor(selectedProfile);
     if(!secret){credentialError();return;}
     token.value='';
     self.action(function(){return api.cfUpdate(secret,selectedProfile.accountId,selectedProfile.project,selectedProfile.kind);});
    }):null;
   var removeRemote=selectedProfile&&selectedProfile.accountId&&selectedProfile.project?node('button',{'type':'button','class':'pol-btn','style':'background:#dc2626!important;color:white!important','click':function(){
    var secret=credentialFor(selectedProfile);
    if(!secret){credentialError();return;}
    if(!window.confirm((self.lang==='fa'?'این استقرار از اکانت Cloudflare حذف شود؟ ':'Delete this deployment from Cloudflare? ')+selectedProfile.host))return;
    token.value='';self.action(function(){return api.cfDelete(secret,selectedProfile.accountId,selectedProfile.project,selectedProfile.kind);}).finally(function(){secret='';});
   }},[icon('trash'),node('span',{},this.lang==='fa'?'حذف از کلادفلر':'Delete from Cloudflare')]):null;
   var profiles=this.s.workerProfiles||[],profileSelect=node('select',{'class':'pol-input','aria-label':this.t('savedWorker'),'change':function(ev){var p=profiles[Number(ev.target.value)];if(!p)return;self.action(function(){return api.save({mode:'worker',host:p.host,uuid:p.uuid,address:p.address||'',port:p.port||443,path:p.path||'/?ed=2560',workerSni:p.workerSni||'',workerDomain:p.customDomain||''});});}},[node('option',{'value':''},this.lang==='fa'?'انتخاب ورکر ذخیره‌شده':'Select a saved Worker')].concat(profiles.map(function(p,i){return node('option',{'value':String(i)},p.host);})));
   var domainControls=null;
   if(selectedProfile&&selectedProfile.accountId&&selectedProfile.project){
    var domain=node('input',{'class':'pol-input','type':'text','dir':'ltr','placeholder':'vpn.example.com','value':selectedProfile.customDomain||'','aria-label':this.t('cfDomain')});
    function domainAction(action){
     var secret=credentialFor(selectedProfile),hostname=domain.value.trim();
     if(!secret){credentialError();return;}
     if(action==='detach'&&!window.confirm(self.lang==='fa'?'دامنه از این استقرار جدا شود؟':'Detach this domain from the deployment?'))return;
     self.action(function(){return api.cfDomain(secret,selectedProfile.accountId,selectedProfile.project,selectedProfile.kind,hostname,action);});
    }
    domainControls=node('details',{},[node('summary',{},this.t('cfDomain')),
     node('div',{'class':'pol-log-tools'},[domain,
      node('button',{'class':'pol-btn','type':'button','click':function(){domainAction('attach');}},this.lang==='fa'?'افزودن دامنه':'Attach domain'),
      node('button',{'class':'pol-btn','type':'button','click':function(){domainAction('detach');}},this.lang==='fa'?'جداکردن دامنه':'Detach domain')
     ]),
     node('small',{},this.lang==='fa'?'پس از فعال‌شدن دامنه در کلادفلر، آن را در فیلد دامنهٔ سفارشی اتصال وارد کنید.':'After Cloudflare activates the domain, enter it in the custom connection domain field.')
    ]);
   }
   var usage=selectedProfile&&this.cfUsageResult&&this.cfUsageResult.host===selectedProfile.host?this.cfUsageResult:null;
   var usageButton=selectedProfile&&selectedProfile.accountId&&selectedProfile.project?this.button('cfUsage',function(){
    var secret=credentialFor(selectedProfile);
    if(!secret){credentialError();return;}
    self.action(function(){return api.cfUsage(secret,selectedProfile.accountId,selectedProfile.project);}).then(function(result){
     if(result){self.cfUsageResult={host:selectedProfile.host,total:result.total,deployment:result.deployment};self.panel();}
    });
   }):null;
   var savedWorker=this.card('savedWorker',this.s.host?[
    node('div',{'style':'display:flex;flex-wrap:wrap;align-items:center;justify-content:space-between;gap:12px'},[node('code',{'dir':'ltr','style':'overflow-wrap:anywhere'},this.s.host),refreshWorker,usageButton,removeWorker,removeRemote]),
    domainControls,
    usage?node('div',{'class':'pol-note'},(this.lang==='fa'?'درخواست‌های استقرار / کل حساب':'Deployment / account requests')+' '+usage.deployment+' / '+usage.total):null,
    node('small',{},this.lang==='fa'?'حذف از پل، دامنه و اطلاعات اتصال این ورکر را از روتر پاک می‌کند. برای حذف خود ورکر از اکانت Cloudflare، آن را در Workers & Pages حذف کنید.':'Removing from Pol clears this Worker connection from the router. To delete the remote Worker, remove it in Cloudflare Workers & Pages.')
   ]:[node('p',{},this.lang==='fa'?'هنوز ورکری انتخاب نشده است.':'No Worker is selected.')]);
   savedWorker.classList.add('pol-saved-worker');savedWorker.querySelector('h3').appendChild(profileSelect);
   var deploymentCard=null;
   if(session||!signInSupported){
    var fields=[cfField('workerName',name),this.listField('workerProxyIps'),this.field('workerFallback')];
    if(!signInSupported)fields.unshift(cfField('deploymentType',deploymentType),cfField('token',token),cfField('accountId',account));
    deploymentCard=this.card('cfTitle',[
     node('small',{},this.t('cfDeployHelp')),
     node('div',{'class':'pol-grid pol-option-grid'},fields),
     this.button('deploy',function(){
      var secret=session?session.token:token.value.trim();
      if(!secret){credentialError();return;}
      token.value='';
      self.action(function(){return api.deploy(secret,session?session.account:account.value,name.value,session?session.kind:deploymentType.value);}).then(function(result){
       if(result&&result.host){self.deployedHost=result.host;self.panel();}
      });
     }),
     this.deployedHost?node('div',{'class':'pol-cf-success','role':'status'},[
      node('strong',{},this.t('deployed')),node('code',{'dir':'ltr'},this.deployedHost),
      this.button('home',function(){self.tab='home';self.draw();})
     ]):null
    ]);
   }
   var guide=node('details',{'class':'pol-card'},[node('summary',{},this.t('tokenLink')),this.cloudflareGuide()]);
   body=[accountCard,savedWorker,deploymentCard,guide].filter(Boolean);
		}else if(this.tab==='routing'){
   this.geoNode=node('div',{'class':'pol-geo-grid'});
   var fa=this.lang==='fa';
   body=[this.card('routeIran',[this.switchField('routeIran'),this.geoWarning=node('div',{'class':'pol-geo-warning','role':'status'}),this.iranDNSField()]),this.geoNode,
    this.card('rebindTitle',[!(this.status.features&&this.status.features.routingLists)?node('p',{},this.t('backendRequired')):null,this.listField('rebindDomains','rebindDomains','example.ir',fa?'فقط برای دامنه‌های مورد اعتماد که پاسخ DNS خصوصی دارند استثنا بسازید. محافظت Rebind برای سایر دامنه‌ها روشن می‌ماند؛ استثناهای موجود روتر حفظ می‌شوند.':'Allow private DNS answers only for trusted domains. Other domains retain Rebind protection; existing router exceptions are preserved.')]),
    this.card('directTitle',[this.listField('directIps','directIps','1.2.3.4 or ext:geoip_IR.dat:ir',fa?'IP، بازه CIDR، geoip:ir یا ext:filename.dat:ir. فایل مرجع باید در پوشه قوانین موجود باشد. IPهای صریح پیش از ورود به تونل مستقیم می‌روند.':'IP, CIDR, geoip:ir or ext:filename.dat:ir. Referenced files must exist in the routing data directory. Literal addresses bypass the tunnel before interception.'),this.listField('bypassDomains','bypass','example.com',fa?'دامنه ساده و زیردامنه‌هایش یا پیشوندهای domain:، full:، regexp: و geosite:؛ این قواعد پیش از مسدودسازی اجرا می‌شوند.':'A hostname and its subdomains, or domain:, full:, regexp: and geosite: expressions. Direct rules take precedence over blocking.')]),
    this.card('block',[node('textarea',{'class':'pol-input pol-code','dir':'ltr','rows':5,'aria-label':this.t('block'),'change':function(ev){self.save('blockDomains',ev.target.value.split(/[\n,]+/).map(function(v){return v.trim();}).filter(Boolean));}},(this.s.blockDomains||[]).join('\n'))])];
		}else if(this.tab==='reports'){
			var search=node('input',{'class':'pol-input pol-log-search','type':'search','value':this.logSearch||'','placeholder':this.t('logSearch'),'aria-label':this.t('logSearch'),'input':function(ev){self.logSearch=ev.target.value;self.paintLogs();}});
			this.logBox=node('pre',{'class':'pol-logs','dir':this.lang==='fa'?'rtl':'ltr','aria-live':'off'},this.t('emptyLogs'));
			body=[this.card('reports',[node('div',{'class':'pol-log-tools'},['all','app','core'].map(function(k){return self.button(k,function(){self.logKind=k;self.panel();},self.logKind===k?'selected':'');}).concat([this.button('refresh',function(){self.fetchLogs();}),this.button(this.pauseLogs?'live':'paused',function(){self.pauseLogs=!self.pauseLogs;self.panel();}),this.button('download',function(){self.download('pol-reports.txt',(self.logLines||[]).join('\n'),'text/plain');}),search])),this.logBox])];
		}else if(this.tab==='updates'){
			this.releaseNode=node('div',{'class':'pol-release'},'');
			body=[this.releaseNode,this.updateProgressNode=node('div',{'class':'pol-update-progress','aria-live':'polite'})];
		}

		dom.content(this.content,body);this.fitFields();this.paintStatus();
		if(this.tab==='reports'){
			if(this.pauseLogs&&this.logKind===this.logLinesKind)this.paintLogs();
			else this.fetchLogs();
		}
		if(this.tab==='updates'){this.renderReleases();this.paintUpdateProgress();}if(this.tab==='routing'){this.renderGeo();if(!this.geo){this.request(api.geo()).then(function(v){self.geo=v;self.renderGeo();}).catch(function(e){self.message(e.message,true);});}}
	},
 formatLog:function(line){
  if(this.lang!=='fa')return line;
  var date=null,text=line.replace(/\b(?:pol-app:\s*)?unknown command\b/gi,'فرمان ناشناخته است'),months={Jan:0,Feb:1,Mar:2,Apr:3,May:4,Jun:5,Jul:6,Aug:7,Sep:8,Oct:9,Nov:10,Dec:11};
  var stamp=/^(?:Mon|Tue|Wed|Thu|Fri|Sat|Sun)\s+([A-Z][a-z]{2})\s+(\d{1,2})\s+(\d{2}):(\d{2}):(\d{2})\s+(\d{4})\s+/.exec(text);
  if(stamp){date=new Date(Number(stamp[6]),months[stamp[1]],Number(stamp[2]),Number(stamp[3]),Number(stamp[4]),Number(stamp[5]));text=text.slice(stamp[0].length);}
  else {stamp=/^(\d{4}-\d{2}-\d{2}T\S+)\s+/.exec(text);if(stamp){date=new Date(stamp[1]);text=text.slice(stamp[0].length);}}
  var rules=[
   [/failed to retrieve response for/gi,'دریافت پاسخ DNS ناموفق بود برای'],[/failed to resolve ip/gi,'تبدیل دامنه به آدرس IP ناموفق بود'],[/returning nil for domain/gi,'پاسخی برای دامنه دریافت نشد:'],[/record not found/gi,'رکورد DNS پیدا نشد'],[/tls: failed to verify certificate/gi,'TLS: اعتبارسنجی گواهی ناموفق بود'],[/x509: cannot validate certificate for (\S+) because it doesn't contain any IP SANs/gi,'گواهی برای IP برابر $1 معتبر نیست؛ گواهی این IP را پوشش نمی‌دهد'],[/failed to dial/gi,'برقراری اتصال ناموفق بود'],[/failed to process outbound traffic/gi,'پردازش ترافیک خروجی ناموفق بود'],[/failed to process request/gi,'پردازش درخواست ناموفق بود'],[/all retry attempts failed/gi,'همه تلاش‌های مجدد ناموفق بودند'],[/remote error/gi,'خطای مقصد'],[/bad handshake/gi,'خطا در برقراری ارتباط اولیه'],[/unexpected EOF/gi,'ارتباط پیش از تکمیل پاسخ بسته شد'],[/from DNS/gi,'از DNS'],[/from /g,'از '],[/Post /g,'درخواست POST به '],
   [/Settings rejected; previous settings restored/g,'تنظیمات پذیرفته نشد؛ تنظیمات قبلی بازیابی شد'],[/Settings applied immediately/g,'تنظیمات اعمال شد'],[/Main button: enabled=true/g,'کلید اصلی روشن شد'],[/Main button: enabled=false/g,'کلید اصلی خاموش شد'],[/Worker deployed:/g,'ورکر ساخته شد:'],[/Registered (\w+) account/g,'اکانت $1 ساخته شد'],[/Transparent routing started on/g,'مسیریابی شبکه فعال شد روی'],[/Updated (geoip|geosite) routing data/g,'فایل مسیریابی $1 به روز شد'],[/Core updated independently to/g,'هسته به این نسخه به روز شد:'],[/Endpoint scan completed/g,'اسکن اتصال پایان یافت'],[/Downloading core/g,'در حال دریافت هسته'],[/Downloading app/g,'در حال دریافت برنامه'],[/Worker hostname and valid UUID are required/g,'دامنه ورکر و شناسه UUID معتبر را وارد کنید'],[/service is disabled/g,'سرویس خاموش است'],[/another background operation is in progress/g,'عملیات دیگری در حال اجرا است'],[/core did not start; see Reports/g,'هسته اجرا نشد؛ گزارش ها را بررسی کنید'],[/connection refused/gi,'اتصال رد شد'],[/connection reset by peer/gi,'اتصال توسط مقصد قطع شد'],[/network is unreachable/gi,'شبکه در دسترس نیست'],[/no route to host/gi,'مسیری به مقصد وجود ندارد'],[/no such host/gi,'دامنه پیدا نشد'],[/context deadline exceeded/gi,'مهلت عملیات پایان یافت'],[/i\/o timeout/gi,'مهلت ارسال یا دریافت پایان یافت'],[/connection timed out/gi,'مهلت اتصال پایان یافت'],[/failed to/g,'خطا در'],[/failed:/g,'ناموفق:'],[/accepted/g,'اتصال پذیرفته شد'],[/rejected/g,'اتصال رد شد'],[/started/g,'اجرا شد'],[/closed/g,'بسته شد'],[/\[Warning\]/gi,'[هشدار]'],[/\[Error\]/gi,'[خطا]'],[/\[Info\]/gi,'[اطلاعات]'],[/\[Debug\]/gi,'[اشکال زدایی]'],[/daemon\.err/g,'خطای سرویس'],[/daemon\.warn/g,'هشدار سرویس'],[/daemon\.info/g,'اطلاعات سرویس'],[/user\.notice/g,'رخداد برنامه'],[/user\.err/g,'خطای برنامه']
  ];rules.forEach(function(rule){text=text.replace(rule[0],rule[1]);});
  return (date&&!isNaN(date.getTime())?new Intl.DateTimeFormat('fa-IR-u-ca-persian',{year:'numeric',month:'2-digit',day:'2-digit',hour:'2-digit',minute:'2-digit',second:'2-digit',hour12:false}).format(date)+' | ':'')+text;
 },

	showConfig:function(){
		var self=this;
		this.request(api.config()).then(function(result){
			var content=JSON.stringify(result.config,null,2);
			var dialog=node('dialog',{'class':'pol-config-dialog','aria-label':self.t('configView'),'close':function(){dialog.remove();}},[
				node('h3',{},self.t('configView')),
				node('pre',{'class':'pol-code','dir':'ltr','tabindex':'0'},content),
				node('div',{'class':'pol-log-tools'},[
					self.button('configDownload',function(){self.download('xray.json',content,'application/json');}),
					node('button',{'type':'button','class':'pol-btn','click':function(){dialog.close();}},self.lang==='fa'?'بستن':'Close')
				])
			]);
			self.root.appendChild(dialog);
			dialog.showModal();
		}).catch(function(e){self.message(e.message,true);});
	},
	paintLogs:function(){
		if(!this.logBox||!this.logBox.isConnected)return;
		var all=this.logLines||[], query=(this.logSearch||'').trim().toLowerCase();
		var lines=query?all.filter(function(line){return line.toLowerCase().includes(query);}):all;
		var content=lines.length?lines.join('\n'):this.t(query?'noLogResults':'emptyLogs');
		if(this.logBox.textContent===content)return;
		var atEnd=this.logBox.scrollHeight-this.logBox.scrollTop-this.logBox.clientHeight<50;
		this.logBox.textContent=content;
		if(atEnd&&!query)this.logBox.scrollTop=this.logBox.scrollHeight;
	},
	fetchLogs:function(){
		var self=this, kind=this.logKind, target=this.logBox;
		if(this.logRequest===target)return;
		this.logRequest=target;
		return this.request(api.logs(kind)).then(function(v){
			if(self.logKind!==kind||self.logBox!==target||!target.isConnected)return;
			self.logLines=(v.lines||[]).map(function(line){return self.formatLog(line);});
			self.logLinesKind=kind;
			self.paintLogs();
		}).catch(function(e){if(target.isConnected)self.message(e.message,true);})
		  .finally(function(){if(self.logRequest===target)self.logRequest=null;});
	},
 paintUpdateProgress:function(){
  if(!this.updateProgressNode)return;var j=this.status.job||{},p=j.progress,fa=this.lang==='fa';
  if(j.method!=='update'||!p){dom.content(this.updateProgressNode,[]);return;}
  var active=j.state==='running'||j.state==='pending',phase=j.state==='failed'?(fa?'ناموفق: ':'Failed: ')+this.updateError(j.error):j.state==='done'?this.t('done'):({downloading:fa?'در حال دانلود':'Downloading',verifying:fa?'بررسی صحت فایل':'Verifying file',installing:fa?'در حال نصب':'Installing'}[p.phase]||this.t('starting'));
  var percent=p.total>0?Math.min(100,Math.floor(p.received*100/p.total)):null,attrs={'max':100,'aria-label':phase};if(percent!==null)attrs.value=percent;
  dom.content(this.updateProgressNode,[node('strong',{},phase),active&&p.phase==='downloading'?node('progress',attrs):null,p.phase==='downloading'?node('span',{'dir':'ltr'},(percent!==null?percent+'% · ':'')+(p.received/1048576).toFixed(2)+' MB'+(p.total>0?' / '+(p.total/1048576).toFixed(2)+' MB':'')):null]);
 },

 checkRelease:function(component){
  var self=this;this.releaseChecking=this.releaseChecking||{};if(this.releaseChecking[component])return;
  this.releaseChecking[component]=true;this.renderReleases();
  this.request(api.releases(component)).then(function(result){self.releaseInfo=self.releaseInfo||{};Object.keys(self.releaseInfo).forEach(function(key){if(key.indexOf(component)===0)delete self.releaseInfo[key];});Object.assign(self.releaseInfo,result);}).catch(function(e){self.releaseInfo=self.releaseInfo||{};self.releaseInfo[component+'Error']=e.message;}).finally(function(){self.releaseChecking[component]=false;if(self.tab==='updates')self.renderReleases();});
 },
 renderReleases:function(){
  if(!this.releaseNode)return;var self=this,r=this.releaseInfo||{},fa=this.lang==='fa';
  dom.content(this.releaseNode,['app','core'].map(function(k){
   var current=k==='app'?self.status.version:((self.status.coreVersion||'').match(/^Xray\s+([^\s]+)/)||[])[1]||'—',checking=!!(self.releaseChecking||{})[k],error=r[k+'Error'],latest=r[k],upToDate=r[k+'UpToDate'];
   var text=checking?(fa?'در حال بررسی…':'Checking…'):error?(error.indexOf('404')>=0?(fa?'هنوز نسخه ای منتشر نشده است.':'No release has been published yet.'):error):upToDate?(fa?'به روز است':'Up to date'):latest?(r[k+'Newer']?(fa?'نسخه جدید: ':'New version: ')+latest:(fa?'مقایسه نسخه ممکن نیست.':'Unable to compare versions.')):(fa?'هنوز بررسی نشده است.':'Not checked yet.');
   var check=self.button('check',function(){self.checkRelease(k);});check.disabled=self.busy||checking;check.dataset.polManaged='release';check.setAttribute('aria-label',(fa?'بررسی ':'Check ')+(k==='core'?'Xray':'Pol'));
   var update=self.button('updateAction',function(){self.action(function(){return api.update(k,latest);},'verifying').then(function(result){if(result&&result.upToDate)self.message(fa?'به روز است':'Up to date');else if(result)self.checkRelease(k);});});update.disabled=self.busy||checking||!!error||!r[k+'Newer'];update.dataset.polManaged='release';
   var repo=k==='core'?'XTLS/Xray-core':'dreamboxone/pol';
   return self.card(k==='app'?'updateTitle':'core',[node('div',{'class':'pol-version-row'},[node('strong',{},k==='core'?'Xray':'Pol'),node('span',{'class':'pol-version-badge','dir':'ltr'},current),node('span',{'class':'pol-version-note'+(upToDate?' is-current':''),'role':'status'},text),node('div',{'class':'pol-version-actions'},[check,upToDate?null:update])]),node('a',{'class':'pol-update-source','href':'https://github.com/'+repo+'/releases','target':'_blank','rel':'noopener noreferrer'},repo)]);
  }));this.paintUpdateProgress();
 },

	download:function(name,text,type){var u=URL.createObjectURL(new Blob([text],{type:type})),a=node('a',{'href':u,'download':name});document.body.appendChild(a);a.click();a.remove();setTimeout(function(){URL.revokeObjectURL(u);},1000);}
});
