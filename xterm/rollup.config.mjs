import resolve from '@rollup/plugin-node-resolve';
import commonjs from '@rollup/plugin-commonjs';

export default {
    input: 'index.mjs',
    output: { file: '../web/terminal/xterm.mjs', format: 'es' },
    plugins: [resolve(), commonjs()]
};
