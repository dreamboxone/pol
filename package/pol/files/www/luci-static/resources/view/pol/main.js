'use strict';
'require view';
'require rpc';
'require poll';
'require dom';

var api = {};
['status', 'settings', 'logs', 'save', 'toggle', 'import', 'register', 'deploy', 'releases', 'update'].forEach(function(name) {
	var params = {logs:['kind'], save:['settings'], toggle:['enabled'], import:['text'], register:['kind'], deploy:['token','account','name'], update:['component','tag']}[name] || [];
	api[name] = rpc.declare({object:'pol', method:name, params:params, reject:true});
});

var words = {
	fa: {
		title:'پل', subtitle:'اتصال امن، کنترل ساده', home:'خانه', connection:'اتصال', cloudflare:'کلادفلر', routing:'مسیریابی', reports:'گزارش', updates:'به‌روزرسانی', about:'درباره',
		main:'دکمه اصلی', on:'روشن', off:'خاموش', connected:'هسته فعال است', disconnected:'هسته خاموش است', starting:'در حال اجرا…', applied:'تغییرات اعمال شد', applying:'در حال اعمال…', error:'خطا',
		intro:'یک اتصال، برای همه دستگاه‌های شبکه', introText:'نوع اتصال را انتخاب کنید و با دکمه اصلی، سرویس را روشن کنید. تغییرات به‌صورت مستقیم اعمال می‌شوند.',
		mode:'نوع اتصال', variant:'روش اتصال', normal:'معمولی', fragment:'فرگمنت', obfuscated:'مبهم‌سازی', worker:'ورکر', warp:'وارپ', masque:'مسک', serverless:'بدون سرور', tor:'تور', psiphon:'سایفون', custom:'کانفیگ واردشده',
		host:'دامنه Worker', uuid:'شناسه UUID', address:'آدرس اتصال / IP تمیز', port:'پورت', path:'مسیر WebSocket', dns:'DNS راه‌دور', lan:'رابط شبکه محلی', transparent:'اتصال تمام دستگاه‌های شبکه', length:'طول فرگمنت', delay:'تاخیر فرگمنت (میلی‌ثانیه)', endpoint:'نقطه اتصال', transport:'انتقال MASQUE', sni:'نام سرور TLS', chain:'پروکسی زنجیره‌ای (URI)', logLevel:'سطح گزارش هسته',
		importTitle:'وارد کردن کانفیگ', importHelp:'لینک VLESS، Trojan، SOCKS یا کانفیگ JSON هسته Xray و تنظیمات خروجی همین برنامه را وارد کنید.', import:'وارد کردن', paste:'کانفیگ را اینجا قرار دهید…', optional:'اختیاری',
		account:'حساب اتصال', registered:'آماده', missing:'ثبت نشده', registerWarp:'ساخت حساب WARP', registerMasque:'ساخت حساب MASQUE', registrationHelp:'ثبت حساب با سرویس Cloudflare انجام می‌شود. دسترسی اینترنت روتر لازم است.',
		cfTitle:'Worker اختصاصی بسازید', cfHelp:'توکن API با مجوز Workers Scripts Edit یا نقش Workers Admin و شناسه حساب Cloudflare را وارد کنید. توکن ذخیره نمی‌شود. ابتدا زیر‌دامنه workers.dev را در پنل Cloudflare فعال کنید.', token:'توکن API', accountId:'شناسه حساب (Account ID)', workerName:'نام جدید Worker', deploy:'ساخت Worker', deployed:'Worker ساخته شد', tokenLink:'ساخت توکن در Cloudflare',
		bypass:'دامنه‌های مستقیم', block:'دامنه‌های مسدود', domainHelp:'هر دامنه در یک خط؛ مانند example.com. نشانی‌های شبکه خصوصی مستقیم عبور می‌کنند.', bypassHelp:'ترافیک این دامنه‌ها بدون تونل عبور می‌کند.',
		all:'همه', app:'برنامه', core:'هسته', refresh:'تازه‌سازی', download:'دریافت گزارش', emptyLogs:'هنوز رخدادی ثبت نشده است.', paused:'توقف نمایش زنده', live:'نمایش زنده',
		updateTitle:'هسته و برنامه، مستقل از هم', updateHelp:'به‌روزرسانی هسته، رابط را تغییر نمی‌دهد. بسته برنامه مطابق مدیر بسته روتر انتخاب می‌شود و هسته را جایگزین نمی‌کند.', current:'نسخه نصب‌شده', latest:'آخرین نسخه', check:'بررسی نسخه‌ها', updateCore:'به‌روزرسانی هسته', updateApp:'به‌روزرسانی برنامه', repository:'مخزن انتشار برنامه (owner/name)', repoHelp:'پس از ساخت مخزن GitHub، این مقدار را تنظیم کنید. تا آن زمان، به‌روزرسانی آنلاین برنامه در دسترس نیست.', noRepo:'مخزن انتشار هنوز تنظیم نشده است.', verifying:'در حال دریافت و بررسی…', scheduled:'نصب برنامه شروع شد؛ پس از چند لحظه صفحه را تازه‌سازی کنید و گزارش را ببینید.', done:'به‌روزرسانی انجام شد',
		proxy:'پروکسی محلی', firewall:'مسیریابی شبکه', active:'فعال', inactive:'غیرفعال', package:'بسته‌بندی', noApply:'اعمال مستقیم تنظیمات', aboutText:'نسخه مخصوص OpenWrt بر پایه BPB-VPN. رابط مدیریتی LuCI با هسته مستقل Xray، پشتیبانی از IPK و APK و زبان فارسی و انگلیسی.',
		license:'مجوز و منابع', localOnly:'دسترسی پروکسی محلی: فقط خود روتر', notice:'حالت بدون سرور با فرگمنت کار می‌کند و IP خروجی را تغییر نمی‌دهد. Tor به SOCKS روی پورت 9050 و Psiphon به SOCKS روی پورت 10829 نیاز دارد؛ این دو سرویس جداگانه نصب می‌شوند.', configNotice:'در وارد کردن JSON، خروجی‌های Xray استفاده می‌شوند؛ ورودی‌ها، DNS و قوانین مسیر از این برنامه ساخته می‌شوند.', manual:'راهنمای اتصال', manualText:'۱. Worker بسازید یا کانفیگ وارد کنید. ۲. نوع اتصال را انتخاب کنید. ۳. دکمه اصلی را روشن کنید.',
		debug:'اشکال‌زدایی', info:'اطلاعات', warning:'هشدار', none:'بدون گزارش', light:'روشن', dark:'تاریک', export:'خروجی تنظیمات', exportHelp:'فایل خروجی شامل کلیدها و شناسه‌های اتصال است؛ آن را محرمانه نگه دارید.',
	},
	en: {
		title:'Pol', subtitle:'Secure connection. Simple control.', home:'Home', connection:'Connection', cloudflare:'Cloudflare', routing:'Routing', reports:'Reports', updates:'Updates', about:'About',
		main:'Main button', on:'On', off:'Off', connected:'Core is running', disconnected:'Core is stopped', starting:'Working…', applied:'Changes applied', applying:'Applying…', error:'Error',
		intro:'One connection, for your whole network', introText:'Choose a connection and use the main button to start the service. Settings take effect immediately.',
		mode:'Connection type', variant:'Connection variant', normal:'Normal', fragment:'Fragment', obfuscated:'Obfuscated', worker:'Worker', warp:'WARP', masque:'MASQUE', serverless:'Serverless', tor:'Tor', psiphon:'Psiphon', custom:'Imported configuration',
		host:'Worker hostname', uuid:'UUID', address:'Connection address / clean IP', port:'Port', path:'WebSocket path', dns:'Remote DNS', lan:'LAN interface', transparent:'Connect every LAN device', length:'Fragment length', delay:'Fragment delay (milliseconds)', endpoint:'Endpoint', transport:'MASQUE transport', sni:'TLS server name', chain:'Chain proxy (URI)', logLevel:'Core log level',
		importTitle:'Import configuration', importHelp:'Paste a VLESS, Trojan or SOCKS link, Xray JSON, or settings exported by this app.', import:'Import', paste:'Paste configuration here…', optional:'Optional',
		account:'Tunnel account', registered:'Ready', missing:'Not registered', registerWarp:'Register WARP account', registerMasque:'Register MASQUE account', registrationHelp:'Registers with Cloudflare. The router needs internet access.',
		cfTitle:'Deploy your own Worker', cfHelp:'Enter a Cloudflare API token with Workers Scripts Edit or Workers Admin permissions and your Account ID. The token is never saved. Set up a workers.dev subdomain in Cloudflare first.', token:'API token', accountId:'Account ID', workerName:'New Worker name', deploy:'Deploy Worker', deployed:'Worker deployed', tokenLink:'Create a Cloudflare token',
		bypass:'Direct domains', block:'Blocked domains', domainHelp:'One domain per line, such as example.com. Private networks bypass the tunnel.', bypassHelp:'These domains connect directly, outside the tunnel.',
		all:'All', app:'Application', core:'Core', refresh:'Refresh', download:'Download logs', emptyLogs:'No events yet.', paused:'Pause live view', live:'Live view',
		updateTitle:'Update the core and app separately', updateHelp:'Core updates leave the interface in place. App updates select the router’s package format and keep the core in place.', current:'Installed', latest:'Latest', check:'Check for updates', updateCore:'Update core', updateApp:'Update application', repository:'Application release repository (owner/name)', repoHelp:'Set this after creating the GitHub repository. Online app updates remain unavailable until then.', noRepo:'No release repository configured yet.', verifying:'Downloading and verifying…', scheduled:'App installation started. Refresh shortly and check Reports.', done:'Update completed',
		proxy:'Local proxy', firewall:'LAN routing', active:'Active', inactive:'Inactive', package:'Package format', noApply:'Settings apply immediately', aboutText:'An OpenWrt adaptation of BPB-VPN. A LuCI interface with an independent Xray core, IPK and APK packaging, and Persian and English languages.',
		license:'License and sources', localOnly:'Local proxy access: router only', notice:'Serverless fragmentation does not change your public IP. Tor needs a SOCKS service on port 9050; Psiphon needs one on port 10829. Install those services separately.', configNotice:'JSON import uses Xray outbounds. This app builds the inbounds, DNS and routing rules.', manual:'Getting connected', manualText:'1. Deploy a Worker or import a configuration. 2. Choose the connection type. 3. Turn on the main button.',
		debug:'Debug', info:'Info', warning:'Warning', none:'No logs', light:'Light', dark:'Dark', export:'Export settings', exportHelp:'The export contains connection keys and identifiers. Keep it private.',
	}
};

return view.extend({
	handleSave:null, handleSaveApply:null, handleReset:null,
	load:function() { return Promise.all([api.settings(),api.status()]); },
	render:function(data) {
		var self=this;
		this.s=data[0]; this.status=data[1]; this.lang=localStorage.getItem('pol-language')||'fa'; this.theme=localStorage.getItem('pol-theme')||'dark'; this.tab='home'; this.logKind='all'; this.busy=false; this.queue=Promise.resolve();
		if (!words[this.lang]) this.lang='fa';
		if (this.s.error) return E('div',{'class':'alert-message warning'},this.s.error);
		if (!document.querySelector('link[data-pol-style]')) document.head.appendChild(E('link',{'rel':'stylesheet','href':L.resource('pol/style.css'),'data-pol-style':'1'}));
		this.root=E('div'); this.draw();
		poll.add(function(){ return api.status().then(function(v){self.status=v; self.paintStatus(); if(self.tab==='reports'&&!self.pauseLogs) return self.fetchLogs();}).catch(function(e){self.message(e.message,true);}); },5);
		return this.root;
	},
	t:function(key){return words[this.lang][key]||key;},
	variants:function(){if(this.s.mode==='worker')return ['normal','fragment'];if(this.s.mode==='warp')return ['normal','obfuscated'];if(this.s.mode==='masque')return ['normal',this.s.transport==='h3'?'obfuscated':'fragment'];if(this.s.mode==='serverless')return ['fragment'];return ['normal'];},
	button:function(key,fn,cls){return E('button',{'class':'pol-btn '+(cls||''),'type':'button','click':fn},this.t(key));},
	message:function(text,error){if(this.flash){this.flash.textContent=text;this.flash.className='pol-flash'+(error?' is-error':''); this.flash.setAttribute('role',error?'alert':'status');}},
	request:function(p){return p.then(function(r){if(r&&r.error)throw new Error(r.error);return r;});},
	waitJob:function(result){
		var self=this;if(!result.job)return Promise.resolve(result);var deadline=Date.now()+600000;
		return new Promise(function(resolve,reject){function check(){api.status().then(function(s){self.status=s;self.paintStatus();var job=s.job||{};if(job.id===result.job&&job.state==='done')return resolve(job.result||{});if(job.id===result.job&&job.state==='failed')return reject(new Error(job.error));if(Date.now()>deadline)return reject(new Error('Operation timed out; check Reports.'));setTimeout(check,1500);}).catch(reject);}check();});
	},
	action:function(fn,label){
		var self=this;if(this.busy)return Promise.resolve();this.busy=true;this.paintStatus();this.message(this.t(label||'starting'));
		return this.queue.then(fn).then(function(r){return self.request(Promise.resolve(r));}).then(function(r){return self.waitJob(r);}).then(function(r){return Promise.all([api.settings(),api.status()]).then(function(v){self.s=v[0];self.status=v[1];self.message(r.scheduled?self.t('scheduled'):self.t('applied'));self.panel();return r;});}).catch(function(e){self.message(e.message,true);}).finally(function(){self.busy=false;self.paintStatus();if(self.tab==='updates')self.renderReleases();});
	},
	save:function(key,value){
		var self=this;this.s[key]=value;this.message(this.t('applying'));
		this.queue=this.queue.catch(function(){}).then(function(){return self.request(api.save(Object.fromEntries([[key,value]])));}).then(function(){self.message(self.t('applied'));return Promise.all([api.settings(),api.status()]);}).then(function(v){self.s=v[0];self.status=v[1];self.paintStatus();}).catch(function(e){self.message(e.message,true);return api.settings().then(function(s){self.s=s;self.panel();});});
		return this.queue;
	},
	field:function(key,label,type,help){
		var self=this, opts={'class':'pol-input','dir':'ltr','type':type||'text','value':this.s[key]==null?'':this.s[key],'change':function(ev){self.save(key,type==='number'?Number(ev.target.value):ev.target.value);}};
		if(type==='number'){opts.min=1;opts.max=65535;}
		return E('label',{'class':'pol-field'},[E('span',{},this.t(label||key)),E('input',opts),help?E('small',{},this.t(help)):null]);
	},
	select:function(key,values,label){
		var self=this, select=E('select',{'class':'pol-input','change':function(ev){self.save(key,ev.target.value).then(function(){self.panel();});}},values.map(function(v){return E('option',{'value':v},self.t(v));}));select.value=this.s[key];return E('label',{'class':'pol-field'},[E('span',{},this.t(label||key)),select]);
	},
	card:function(title,children){return E('section',{'class':'pol-card'},[E('h3',{},this.t(title))].concat(children));},
	draw:function(){
		var self=this;this.root.className='pol '+(this.theme==='dark'?'pol-dark':'');this.root.dir=this.lang==='fa'?'rtl':'ltr';this.root.lang=this.lang;
		this.flash=E('div',{'class':'pol-flash','aria-live':'polite'});
		this.mainButton=this.button('main',function(){self.action(function(){return api.toggle(!self.status.enabled);});},'pol-power');
		this.statusNode=E('span',{'class':'pol-status'});this.content=E('div',{'class':'pol-content'});
		var language=E('select',{'class':'pol-language','aria-label':'Language','change':function(ev){self.lang=ev.target.value;localStorage.setItem('pol-language',self.lang);self.draw();}},[E('option',{'value':'fa'},'فارسی'),E('option',{'value':'en'},'English')]);language.value=this.lang;
		var theme=this.button(this.theme==='dark'?'light':'dark',function(){self.theme=self.theme==='dark'?'light':'dark';localStorage.setItem('pol-theme',self.theme);self.draw();},'glass');
		dom.content(this.root,[E('header',{'class':'pol-hero'},[E('div',{'class':'pol-brand'},[E('div',{'class':'pol-logo','aria-hidden':'true'},'P'),E('div',{},[E('h2',{},[this.t('title'),E('span',{'class':'pol-version'},'v'+(this.status.version||''))]),E('p',{},this.t('subtitle'))])]),E('div',{'class':'pol-tools'},[language,theme])]),E('nav',{'class':'pol-nav','aria-label':this.t('title')},['home','connection','cloudflare','routing','reports','updates','about'].map(function(key){return self.button(key,function(){self.tab=key;self.draw();},self.tab===key?'selected':'');})),this.flash,this.content,E('footer',{},this.t('noApply'))]);
		this.panel();this.paintStatus();
	},
	paintStatus:function(){
		if(!this.mainButton)return;
		if(this.content)this.content.querySelectorAll('input,select,textarea').forEach(function(el){el.disabled=this.busy;},this);
		this.mainButton.disabled=this.busy;this.mainButton.className='pol-btn pol-power '+(this.status.running?'is-on':'');this.mainButton.textContent='⏻  '+this.t('main')+' · '+this.t(this.status.running?'on':'off');this.mainButton.setAttribute('aria-pressed',String(!!this.status.running));
		this.statusNode.textContent=this.t(this.status.running?'connected':'disconnected');this.statusNode.className='pol-status '+(this.status.running?'is-on':'');
		if(this.stats)dom.content(this.stats,this.statCards());
	},
	statCards:function(){var self=this;return [['core',this.status.coreVersion],['firewall',this.t(this.status.firewall?'active':'inactive')],['package',(this.status.packageManager||'').toUpperCase()]].map(function(v){return E('div',{'class':'pol-stat'},[E('small',{},self.t(v[0])),E('b',{'dir':'auto'},v[1]||'—')]);});},
	panel:function(){
		var self=this,body=[];
		if(this.tab==='home'){
			this.stats=E('div',{'class':'pol-stats'},this.statCards());
			body=[E('section',{'class':'pol-connect'},[E('span',{'class':'pol-eyebrow'},'POL / OPENWRT'),E('h2',{},this.t('intro')),E('p',{},this.t('introText')),E('div',{'class':'pol-power-row'},[this.mainButton,this.statusNode])]),this.stats,this.card('connection',[E('div',{'class':'pol-grid'},[this.select('mode',['worker','warp','masque','serverless','tor','psiphon','custom']),this.select('variant',this.variants())]),E('p',{'class':'pol-note'},this.t('notice'))]),this.card('manual',[E('p',{},this.t('manualText'))])];
		}else if(this.tab==='connection'){
			var fields=[this.select('mode',['worker','warp','masque','serverless','tor','psiphon','custom']),this.select('variant',this.variants())];
			if(this.s.mode==='worker')fields=fields.concat([this.field('host'),this.field('uuid',null,'password'),this.field('address',null,'text','optional'),this.field('port',null,'number'),this.field('path')]);
			if(this.s.mode==='warp'||this.s.mode==='masque'){
				fields.push(this.field('endpoint'));
				if(this.s.mode==='masque')fields.push(this.select('transport',['h2','h3']),this.field('sni'));
			}
			fields=fields.concat([this.field('fragmentLength','length'),this.field('fragmentDelay','delay'),this.field('dns'),this.field('lan'),this.field('chain',null,'password','optional'),this.select('logLevel',['debug','info','warning','error','none'])]);
			var toggle=E('input',{'type':'checkbox','change':function(ev){self.save('transparent',ev.target.checked);}});toggle.checked=this.s.transparent;
			var importText=E('textarea',{'class':'pol-input pol-code','dir':'ltr','rows':7,'placeholder':this.t('paste'),'aria-label':this.t('importTitle')});
			body=[this.card('connection',[E('div',{'class':'pol-grid'},fields),E('label',{'class':'pol-switch'},[toggle,E('span',{},this.t('transparent'))])]),this.card('account',[E('div',{'class':'pol-account-row'},[E('p',{},'WARP: '+this.t(this.s.warp?'registered':'missing')),this.button('registerWarp',function(){self.action(function(){return api.register('warp');});})]),E('div',{'class':'pol-account-row'},[E('p',{},'MASQUE: '+this.t(this.s.masque?'registered':'missing')),this.button('registerMasque',function(){self.action(function(){return api.register('masque');});})]),E('small',{},this.t('registrationHelp'))]),this.card('importTitle',[E('p',{},this.t('importHelp')),importText,E('small',{},this.t('configNotice')),this.button('import',function(){self.action(function(){return api.import(importText.value);});})])];
		}else if(this.tab==='cloudflare'){
			var token=E('input',{'class':'pol-input','type':'password','dir':'ltr','autocomplete':'off','spellcheck':'false'}),account=E('input',{'class':'pol-input','dir':'ltr','autocomplete':'off'}),name=E('input',{'class':'pol-input','dir':'ltr','value':'pol-'+Date.now().toString(36).slice(-7)});
			function cfField(key,input){return E('label',{'class':'pol-field'},[E('span',{},self.t(key)),input]);}
			body=[this.card('cfTitle',[E('p',{},this.t('cfHelp')),E('a',{'href':'https://dash.cloudflare.com/profile/api-tokens','target':'_blank','rel':'noopener noreferrer'},this.t('tokenLink')),E('div',{'class':'pol-grid'},[cfField('token',token),cfField('accountId',account),cfField('workerName',name)]),this.button('deploy',function(){var secret=token.value;token.value='';self.action(function(){return api.deploy(secret,account.value,name.value);}).finally(function(){secret='';});})])];
		}else if(this.tab==='routing'){
			var domainField=function(key,label){return E('label',{'class':'pol-field'},[E('span',{},self.t(label)),E('textarea',{'class':'pol-input pol-code','dir':'ltr','rows':9,'change':function(ev){self.save(key,ev.target.value.split(/[\n,]+/).map(function(v){return v.trim();}).filter(Boolean));}},(self.s[key]||[]).join('\n'))]);};
			body=[this.card('routing',[E('p',{},this.t('domainHelp')),E('div',{'class':'pol-grid'},[domainField('bypassDomains','bypass'),domainField('blockDomains','block')]),E('small',{},this.t('bypassHelp'))])];
		}else if(this.tab==='reports'){
			this.logBox=E('pre',{'class':'pol-logs','dir':'ltr','aria-live':'off'},this.t('emptyLogs'));
			body=[this.card('reports',[E('div',{'class':'pol-log-tools'},['all','app','core'].map(function(k){return self.button(k,function(){self.logKind=k;self.panel();},self.logKind===k?'selected':'');}).concat([this.button('refresh',function(){self.fetchLogs();}),this.button(this.pauseLogs?'live':'paused',function(){self.pauseLogs=!self.pauseLogs;self.panel();}),this.button('download',function(){self.download('pol-reports.txt',(self.logLines||[]).join('\n'),'text/plain');})])),this.logBox])];
		}else if(this.tab==='updates'){
			this.releaseNode=E('div',{'class':'pol-release'},'');
			body=[this.card('updateTitle',[E('p',{},this.t('updateHelp')),E('div',{'class':'pol-stats'},this.statCards()),this.button('check',function(){self.message(self.t('starting'));self.request(api.releases()).then(function(v){self.releaseInfo=v;self.renderReleases();self.message('');}).catch(function(e){self.message(e.message,true);});}),this.releaseNode]),this.card('repository',[this.field('releaseRepository','repository','text','repoHelp')])];
		}else{
			body=[this.card('about',[E('p',{},this.t('aboutText')),E('p',{},this.t('notice')),E('p',{},this.t('localOnly')),E('h4',{},this.t('license')),E('p',{},'GPL-3.0-only · Xray-core: MPL-2.0 · Vazirmatn: OFL-1.1'),E('a',{'href':'https://github.com/bia-pain-bache/BPB-VPN','target':'_blank','rel':'noopener noreferrer'},'BPB-VPN'),E('p',{},this.t('exportHelp')),this.button('export',function(){self.download('pol-settings.json',JSON.stringify(self.s,null,2),'application/json');})])];
		}
		dom.content(this.content,body);this.paintStatus();
		if(this.tab==='reports')this.fetchLogs();if(this.tab==='updates')this.renderReleases();
	},
	fetchLogs:function(){var self=this;return this.request(api.logs(this.logKind)).then(function(v){self.logLines=v.lines||[];if(self.logBox){var atEnd=self.logBox.scrollHeight-self.logBox.scrollTop-self.logBox.clientHeight<50;self.logBox.textContent=self.logLines.length?self.logLines.join('\n'):self.t('emptyLogs');if(atEnd)self.logBox.scrollTop=self.logBox.scrollHeight;}}).catch(function(e){self.message(e.message,true);});},
	renderReleases:function(){
		if(!this.releaseNode)return;var self=this,r=this.releaseInfo||{},items=[];
		['core','app'].forEach(function(k){items.push(E('div',{'class':'pol-update-row'},[E('div',{},[E('b',{},self.t(k)),E('small',{},self.t('latest')+': '+(r[k]||r[k+'Error']||(k==='app'?self.t('noRepo'):'—')))]),self.button(k==='core'?'updateCore':'updateApp',function(){self.action(function(){return api.update(k,r[k]||'');},'verifying');})]));});
		dom.content(this.releaseNode,items);this.releaseNode.querySelectorAll('button').forEach(function(btn,i){btn.disabled=this.busy||!(i===0?r.core:r.app);},this);
	},
	download:function(name,text,type){var u=URL.createObjectURL(new Blob([text],{type:type})),a=E('a',{'href':u,'download':name});document.body.appendChild(a);a.click();a.remove();setTimeout(function(){URL.revokeObjectURL(u);},1000);}
});
