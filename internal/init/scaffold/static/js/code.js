function copyCode(btn) {
    var code = btn.closest('.code-block').querySelector('code').textContent;
    var textSpan = btn.querySelector('.copy-text');
    var finish = function (ok) {
        textSpan.textContent = ok ? 'Copied!' : 'Failed';
        btn.classList.toggle('copied', ok);
        setTimeout(function () {
            textSpan.textContent = 'Copy';
            btn.classList.remove('copied');
        }, ok ? 2000 : 1500);
    };
    var fallback = function () {
        var area = document.createElement('textarea');
        area.value = code;
        area.style.position = 'fixed';
        area.style.opacity = '0';
        document.body.appendChild(area);
        area.select();
        var ok = document.execCommand('copy');
        area.remove();
        finish(ok);
    };

    if (navigator.clipboard && window.isSecureContext) {
        navigator.clipboard.writeText(code).then(function () { finish(true); }, fallback);
        return;
    }
    fallback();
}
