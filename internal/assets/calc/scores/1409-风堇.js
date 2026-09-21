const scoreRule=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.default = default_1;
function default_1({ cons, rule, def }) {
    let particularAttr = { ...usefulAttr['风堇'] };
    if (cons == 6) {
        particularAttr.cdmg = 100;
        return rule(`风堇-满命`, particularAttr);
    }
    return def(usefulAttr['风堇']);
}

return exports;})();
