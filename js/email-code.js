/* ================================================================
   Signup email verification (Gmail only).
   The first submit of a signup form asks the server to email a 6-digit code
   and shows a code field above the submit button; the next submit returns the
   typed code so it can be sent with POST /api/auth/signup.
   ================================================================ */
(function (window) {
    'use strict';

    async function post(path, payload) {
        let res;
        try {
            res = await fetch(path, {
                method: 'POST',
                credentials: 'same-origin',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(payload)
            });
        } catch (_) {
            throw new Error('Could not reach the server. Please try again.');
        }
        const data = await res.json().catch(() => ({}));
        if (!res.ok) throw new Error(data.error || 'Something went wrong. Please try again.');
        return data;
    }

    function buildBox(inputClass) {
        const box = document.createElement('div');
        box.className = 'ic-email-code';
        box.style.cssText = 'margin:12px 0 4px;';

        const hint = document.createElement('p');
        hint.className = 'ic-email-code-hint';
        hint.style.cssText = 'font-size:13px;line-height:1.45;margin:0 0 8px;opacity:.85;';

        const input = document.createElement('input');
        input.type = 'text';
        input.inputMode = 'numeric';
        input.autocomplete = 'one-time-code';
        input.maxLength = 6;
        input.placeholder = '6-digit code';
        input.setAttribute('aria-label', 'Verification code');
        input.className = inputClass || '';
        input.style.cssText = 'letter-spacing:6px;text-align:center;font-size:18px;';
        input.addEventListener('input', () => { input.value = input.value.replace(/\D/g, '').slice(0, 6); });

        const resend = document.createElement('button');
        resend.type = 'button';
        resend.className = 'ic-email-code-resend';
        resend.textContent = 'Resend code';
        resend.style.cssText = 'background:none;border:0;padding:6px 0;color:inherit;text-decoration:underline;cursor:pointer;font-size:13px;';
        resend.addEventListener('click', async () => {
            resend.disabled = true;
            try {
                await post('/api/auth/signup-code', { email: box.dataset.email, role: box.dataset.role });
                hint.textContent = `We sent a new code to ${box.dataset.email}.`;
            } catch (err) {
                hint.textContent = err.message;
            } finally {
                setTimeout(() => { resend.disabled = false; }, 2000);
            }
        });

        box.append(hint, input, resend);
        return box;
    }

    // Returns the code to send with the signup request, or '' after emailing a
    // new code that the person still has to type in. Throws with a readable message.
    async function signupCode({ form, email, role, before, inputClass }) {
        const normalized = String(email || '').trim().toLowerCase();
        if (!/@(gmail|googlemail)\.com$/.test(normalized)) {
            throw new Error('Please sign up with a Gmail address (name@gmail.com).');
        }
        let box = form.querySelector('.ic-email-code');
        if (box && box.dataset.email === normalized && box.dataset.role === role) {
            const input = box.querySelector('input');
            if (input.value.length === 6) return input.value;
            input.focus();
            throw new Error(`Enter the 6-digit code we sent to ${normalized}.`);
        }

        if (before) before.disabled = true;
        try {
            await post('/api/auth/signup-code', { email: normalized, role });
        } finally {
            if (before) before.disabled = false;
        }
        if (!box) {
            box = buildBox(inputClass);
            before.parentNode.insertBefore(box, before);
        }
        box.dataset.email = normalized;
        box.dataset.role = role;
        box.querySelector('.ic-email-code-hint').textContent =
            `We sent a 6-digit code to ${normalized}. Enter it below, then submit again to finish creating your account.`;
        const input = box.querySelector('input');
        input.value = '';
        input.focus();
        return '';
    }

    window.ICEmailCode = { signupCode };
})(window);
