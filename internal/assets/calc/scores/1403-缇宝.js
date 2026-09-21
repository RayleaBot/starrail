const scoreRule=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.default = default_1;
function default_1({ attr, rule, def }) {
    let particularAttr = { ...usefulAttr['缇宝'] };
    if (attr.speed > 109) {
        particularAttr.speed = 100;
        return rule(`缇宝-高速`, particularAttr);
    }
    return def(usefulAttr['缇宝']);
}

return exports;})();
