(function () {
    function getMenuElements() {
        return {
            button: document.querySelector('.header-menu-btn'),
            nav: document.querySelector('.header-nav')
        };
    }

    function syncMenu(open) {
        var parts = getMenuElements();
        if (!parts.button || !parts.nav) {
            return;
        }

        parts.button.classList.toggle('open', open);
        parts.button.setAttribute('aria-expanded', open ? 'true' : 'false');
        parts.nav.classList.toggle('open', open);
    }

    window.toggleMenu = function toggleMenu() {
        var parts = getMenuElements();
        if (!parts.button) {
            return;
        }
        syncMenu(!parts.button.classList.contains('open'));
    };

    function closeMenu() {
        syncMenu(false);
    }

    document.addEventListener('click', function (event) {
        var parts = getMenuElements();
        if (!parts.button || !parts.nav) {
            return;
        }

        if (event.target.closest('.header-menu-btn')) {
            return;
        }

        if (!event.target.closest('.header-nav')) {
            closeMenu();
            return;
        }

        if (event.target.closest('.header-nav a, .header-nav .theme-toggle')) {
            closeMenu();
        }
    });

    document.addEventListener('keydown', function (event) {
        if (event.key === 'Escape') {
            closeMenu();
        }
    });

    window.addEventListener('resize', function () {
        if (window.innerWidth > 768) {
            closeMenu();
        }
    });
})();
