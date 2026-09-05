// 降雪アニメ。外部ライブラリに依存せず canvas だけで描く。
// 69MB の wasm を待つ間ずっと動くので軽く保つ。
window.Snow = {
    // canvasId の canvas へ降雪を描き始める。制御用に stop を持つオブジェクトを返す
    start(canvasId) {
        const canvas = document.getElementById(canvasId);
        const ctx = canvas.getContext("2d");
        let flakes = [];
        let raf = 0;
        function spawn() {
            return {
                x: Math.random() * canvas.width,
                y: Math.random() * canvas.height,
                s: Math.random() < 0.35 ? 3 : 2, // 粒サイズ。ドット雪の四角
                vy: Math.random() * 0.6 + 0.25,
                vx: Math.random() * 0.4 - 0.2,
                a: Math.random() * 0.5 + 0.3,
            };
        }
        function resize() {
            canvas.width = window.innerWidth;
            canvas.height = window.innerHeight;
            ctx.imageSmoothingEnabled = false; // 補間を切りドット感を保つ
            const count = Math.floor((canvas.width * canvas.height) / 9000);
            flakes = Array.from({ length: count }, spawn);
        }
        function frame() {
            ctx.clearRect(0, 0, canvas.width, canvas.height);
            for (const f of flakes) {
                f.y += f.vy;
                f.x += f.vx;
                if (f.y > canvas.height) {
                    f.y = -4;
                    f.x = Math.random() * canvas.width;
                }
                ctx.fillStyle = "rgba(222,236,246," + f.a + ")";
                ctx.fillRect(Math.round(f.x), Math.round(f.y), f.s, f.s);
            }
            raf = requestAnimationFrame(frame);
        }
        window.addEventListener("resize", resize);
        resize();
        frame();
        return {
            stop() {
                cancelAnimationFrame(raf);
                window.removeEventListener("resize", resize);
            },
        };
    },
};
