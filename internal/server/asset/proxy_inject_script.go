// Copyright (c) 2026 DYCloud J.
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED.

package asset

import (
	"fmt"
	"strings"
)

// ProxyInterceptScript 生成浏览器端拦截脚本
// func GenerateProxyInterceptScript(proxyPrefix string, token string, debug bool) string {
// 	// 移除尾部斜杠（脚本中会自动添加）
// 	proxyPrefix = strings.TrimSuffix(proxyPrefix, "/")

// 	debugStr := "false"
// 	if debug {
// 		debugStr = "true"
// 	}

// 	script := fmt.Sprintf(`
// (function() {
//   'use strict';

//   const PROXY_PREFIX = '%s';
//   const TOKEN = '%s'; // 新增：令牌常量
//   const DEBUG = %s;

//   // 日志函数
//   function log(...args) {
//     if (DEBUG) {
//       console.log('[OpsHub Proxy]', ...args);
//     }
//   }

//   // URL 重写函数
//   function rewriteURL(url) {
//     if (!url || typeof url !== 'string') return url;

//     // 跳过外部链接
//     if (url.startsWith('http://') || url.startsWith('https://') || url.startsWith('//')) {
//       return url;
//     }

//     // 跳过特殊协议
//     if (url.startsWith('data:') || url.startsWith('blob:') || url.startsWith('javascript:') ||
//         url.startsWith('mailto:') || url.startsWith('tel:') || url.startsWith('#')) {
//       return url;
//     }

//     // 跳过已经包含代理前缀的
//     if (url.startsWith(PROXY_PREFIX)) {
//       return url;
//     }

//     // 使用浏览器原生 URL 解析（处理绝对路径和相对路径）
//     try {
//       // 基于当前页面 URL 解析相对路径
//       const absoluteUrl = new URL(url, window.location.href);

//       // 只处理同源的 URL
//       if (absoluteUrl.origin === window.location.origin) {
//         const pathname = absoluteUrl.pathname;

//         // 如果路径已经包含代理前缀，直接返回
//         if (pathname.startsWith(PROXY_PREFIX)) {
//           return absoluteUrl.href;
//         }

//         // 添加代理前缀
//         absoluteUrl.pathname = PROXY_PREFIX + pathname;
//         // ============== 核心：自动添加 token 查询参数 ==============
//         absoluteUrl.searchParams.set('token', TOKEN);
//         return absoluteUrl.href;
//       }
//     } catch (e) {
//       // URL 解析失败，使用简单的字符串拼接
//       if (url.startsWith('/')) {
//         // return PROXY_PREFIX + url;
//         let rewrittenUrl = PROXY_PREFIX + url;
//         // 兼容已有查询参数
//         rewrittenUrl += rewrittenUrl.includes('?') ? '&token=' : '?token=';
//         rewrittenUrl += encodeURIComponent(TOKEN);
//         return rewrittenUrl;
//       }
//     }

//     // 其他情况保持不变
//     return url;
//   }

//   // ==================== 拦截 document.createElement ====================
//   // 这个必须最先执行，在任何脚本加载之前
//   const originalCreateElement = document.createElement;
//   document.createElement = function(tagName, options) {
//     const element = originalCreateElement.call(this, tagName, options);

//     // 拦截 script 标签的 src 属性设置
//     if (tagName.toLowerCase() === 'script') {
//       const originalSrcDescriptor = Object.getOwnPropertyDescriptor(HTMLScriptElement.prototype, 'src');
//       Object.defineProperty(element, 'src', {
//         get: function() {
//           return originalSrcDescriptor.get.call(this);
//         },
//         set: function(value) {
//           // 跳过 module 类型的 script（Vite/Webpack 动态 import）
//           // 这些由浏览器原生处理，不需要重写
//           if (this.type === 'module' || this.getAttribute('type') === 'module') {
//             originalSrcDescriptor.set.call(this, value);
//             return;
//           }

//           const rewrittenValue = rewriteURL(value);
//           log('Rewrite script.src:', rewrittenValue);
//           originalSrcDescriptor.set.call(this, rewrittenValue);
//         }
//       });
//     }

//     // 拦截 link 标签的 href 属性设置
//     else if (tagName.toLowerCase() === 'link') {
//       const originalHrefDescriptor = Object.getOwnPropertyDescriptor(HTMLLinkElement.prototype, 'href');
//       Object.defineProperty(element, 'href', {
//         get: function() {
//           return originalHrefDescriptor.get.call(this);
//         },
//         set: function(value) {
//           const rewrittenValue = rewriteURL(value);
//           log('Rewrite link.href:', rewrittenValue);
//           originalHrefDescriptor.set.call(this, rewrittenValue);
//         }
//       });
//     }

//     // 拦截 img 标签的 src 属性设置
//     else if (tagName.toLowerCase() === 'img') {
//       const originalSrcDescriptor = Object.getOwnPropertyDescriptor(HTMLImageElement.prototype, 'src');
//       Object.defineProperty(element, 'src', {
//         get: function() {
//           return originalSrcDescriptor.get.call(this);
//         },
//         set: function(value) {
//           const rewrittenValue = rewriteURL(value);
//           log('Rewrite img.src:', rewrittenValue);
//           originalSrcDescriptor.set.call(this, rewrittenValue);
//         }
//       });
//     }

//     return element;
//   };

//   // ==================== 拦截 fetch ====================
//   if (window.fetch) {
//     const originalFetch = window.fetch;
//     window.fetch = function(input, init) {
//       let url = input;

//       if (typeof input === 'string') {
//         url = rewriteURL(input);
//       } else if (input instanceof Request) {
//         const originalUrl = input.url;
//         const rewrittenUrl = rewriteURL(originalUrl);
//         if (originalUrl !== rewrittenUrl) {
//           input = new Request(rewrittenUrl, input);
//           url = rewrittenUrl;
//         }
//       }

//       log('fetch:', url);
//       return originalFetch.call(this, input, init);
//     };
//   }

//   // ==================== 拦截 XMLHttpRequest ====================
//   if (window.XMLHttpRequest) {
//     const originalXHROpen = XMLHttpRequest.prototype.open;
//     XMLHttpRequest.prototype.open = function(method, url, ...args) {
//       const rewrittenUrl = rewriteURL(url);
//       log('XHR:', method, rewrittenUrl);
//       return originalXHROpen.call(this, method, rewrittenUrl, ...args);
//     };
//   }

//   // ==================== 拦截动态 import ====================
//   // 注意：这个方法可能不适用于所有打包工具
//   if (window.importScripts) {
//     const originalImportScripts = window.importScripts;
//     window.importScripts = function(...urls) {
//       const rewrittenUrls = urls.map(url => rewriteURL(url));
//       log('importScripts:', rewrittenUrls);
//       return originalImportScripts.apply(this, rewrittenUrls);
//     };
//   }

//   // ==================== 拦截动态创建的元素（备用） ====================
//   const observer = new MutationObserver(function(mutations) {
//     mutations.forEach(function(mutation) {
//       mutation.addedNodes.forEach(function(node) {
//         if (node.nodeType !== 1) return; // 只处理元素节点

//         try {
//           // 处理 script 标签（备用，主要由 createElement 拦截处理）
//           if (node.tagName === 'SCRIPT' && node.src && !node.src.startsWith(PROXY_PREFIX)) {
//             const originalSrc = node.src;
//             const rewrittenSrc = rewriteURL(originalSrc);
//             if (originalSrc !== rewrittenSrc) {
//               node.src = rewrittenSrc;
//               log('Rewrite script src (observer):', rewrittenSrc);
//             }
//           }

//           // 处理 img 标签
//           else if (node.tagName === 'IMG' && node.src && !node.src.startsWith(PROXY_PREFIX)) {
//             const originalSrc = node.src;
//             const rewrittenSrc = rewriteURL(originalSrc);
//             if (originalSrc !== rewrittenSrc) {
//               node.src = rewrittenSrc;
//               log('Rewrite img src (observer):', rewrittenSrc);
//             }
//           }

//           // 处理 link 标签
//           else if (node.tagName === 'LINK' && node.href && !node.href.startsWith(PROXY_PREFIX)) {
//             const originalHref = node.href;
//             const rewrittenHref = rewriteURL(originalHref);
//             if (originalHref !== rewrittenHref) {
//               node.href = rewrittenHref;
//               log('Rewrite link href (observer):', rewrittenHref);
//             }
//           }

//           // 处理 iframe 标签
//           else if (node.tagName === 'IFRAME' && node.src && !node.src.startsWith(PROXY_PREFIX)) {
//             const originalSrc = node.src;
//             const rewrittenSrc = rewriteURL(originalSrc);
//             if (originalSrc !== rewrittenSrc) {
//               node.src = rewrittenSrc;
//               log('Rewrite iframe src (observer):', rewrittenSrc);
//             }
//           }

//           // 处理 video/audio 标签
//           else if ((node.tagName === 'VIDEO' || node.tagName === 'AUDIO') && node.src && !node.src.startsWith(PROXY_PREFIX)) {
//             const originalSrc = node.src;
//             const rewrittenSrc = rewriteURL(originalSrc);
//             if (originalSrc !== rewrittenSrc) {
//               node.src = rewrittenSrc;
//               log('Rewrite media src (observer):', rewrittenSrc);
//             }
//           }
//         } catch (e) {
//           if (DEBUG) console.error('[OpsHub Proxy] Error rewriting element:', e);
//         }
//       });
//     });
//   });

//   // 开始观察
//   observer.observe(document.documentElement, {
//     childList: true,
//     subtree: true
//   });

//   // ==================== 拦截 History API ====================
//   if (window.history) {
//     const originalPushState = history.pushState;
//     const originalReplaceState = history.replaceState;

//     history.pushState = function(state, title, url) {
//       if (url) {
//         url = rewriteURL(url);
//         log('pushState:', url);
//       }
//       return originalPushState.call(this, state, title, url);
//     };

//     history.replaceState = function(state, title, url) {
//       if (url) {
//         url = rewriteURL(url);
//         log('replaceState:', url);
//       }
//       return originalReplaceState.call(this, state, title, url);
//     };
//   }

//   // ==================== 拦截 WebSocket ====================
//   if (window.WebSocket) {
//     const originalWebSocket = window.WebSocket;
//     window.WebSocket = function(url, protocols) {
//       const rewrittenUrl = rewriteURL(url);
//       log('WebSocket:', rewrittenUrl);
//       return new originalWebSocket(rewrittenUrl, protocols);
//     };
//     // 保留原型链
//     window.WebSocket.prototype = originalWebSocket.prototype;
//   }

//   // ==================== 拦截 EventSource ====================
//   if (window.EventSource) {
//     const originalEventSource = window.EventSource;
//     window.EventSource = function(url, config) {
//       const rewrittenUrl = rewriteURL(url);
//       log('EventSource:', rewrittenUrl);
//       return new originalEventSource(rewrittenUrl, config);
//     };
//     // 保留原型链
//     window.EventSource.prototype = originalEventSource.prototype;
//   }

//   log('Intercept script loaded successfully');
// })();
// `, proxyPrefix, token, debugStr)

// 	return script
// }

// ProxyInterceptScript 生成浏览器端拦截脚本
// 新增 token 参数：需要携带的令牌值

// ProxyInterceptScript 生成浏览器端拦截脚本
func GenerateProxyInterceptScript(proxyPrefix string, token string, debug bool) string {
	// 移除尾部斜杠（脚本中会自动添加）
	proxyPrefix = strings.TrimSuffix(proxyPrefix, "/")

	debugStr := "false"
	if debug {
		debugStr = "true"
	}

	script := fmt.Sprintf(`
(function() {
  'use strict';

  const PROXY_PREFIX = '%s';
  const TOKEN = '%s';
  const DEBUG = %s;
  const current = new URL(window.location.href);
  const PAGE_TOKEN = current.searchParams.get('token') || TOKEN;
  const PROXY_PREFIX_WITH_SLASH = PROXY_PREFIX.endsWith('/') ? PROXY_PREFIX : PROXY_PREFIX + '/';

  // 日志函数
  function log(...args) {
    if (DEBUG) {
      console.log('[OpsHub Proxy]', ...args);
    }
  }

  function withToken(urlObj) {
    urlObj.searchParams.set('token', PAGE_TOKEN);
    return urlObj;
  }

  function joinProxyPath(pathname) {
    return PROXY_PREFIX_WITH_SLASH + pathname.replace(/^\//, '');
  }

  // ==================== 修复核心：重写URL逻辑（解决静态资源不生效） ====================
  function rewriteURL(url) {
    if (!url || typeof url !== 'string') return url;

    // 跳过特殊协议（data/blob/mailto等）
    if (url.startsWith('data:') || url.startsWith('blob:') || url.startsWith('javascript:') ||
        url.startsWith('mailto:') || url.startsWith('tel:') || url.startsWith('#')) {
      return url;
    }

    try {
      // 统一解析为URL对象（兼容相对路径/绝对路径）
      const absoluteUrl = new URL(url, window.location.href);

      // 只跳过跨域链接，同源链接强制处理
      if (absoluteUrl.origin !== window.location.origin) {
        return url;
      }

      // 已经是代理路径，只补 token
      if (absoluteUrl.pathname === PROXY_PREFIX || absoluteUrl.pathname.startsWith(PROXY_PREFIX_WITH_SLASH)) {
        withToken(absoluteUrl);
        return absoluteUrl.href;
      }

      // 拼接代理前缀 + 强制添加 token
      absoluteUrl.pathname = joinProxyPath(absoluteUrl.pathname);
      withToken(absoluteUrl);

      log('重写URL成功:', url, '→', absoluteUrl.href);
      return absoluteUrl.href;

    } catch (e) {
      // 降级处理：路径以/开头，直接拼接前缀和 token
      if (url.startsWith('/')) {
        const rewritten = new URL(joinProxyPath(url), window.location.origin);
        withToken(rewritten);
        return rewritten.pathname + rewritten.search + rewritten.hash;
      }
    }

    return url;
  }

  // ==================== 拦截 document.createElement（动态创建元素） ====================
  const originalCreateElement = document.createElement;
  document.createElement = function(tagName, options) {
    const element = originalCreateElement.call(this, tagName, options);
    const lowerTag = tagName.toLowerCase();

    // 统一处理需要拦截的资源标签
    const resourceTags = ['script', 'link', 'img', 'iframe', 'video', 'audio'];
    if (resourceTags.includes(lowerTag)) {
      const attr = lowerTag === 'link' ? 'href' : 'src';
      const prototype = element.__proto__;
      const originalDescriptor = Object.getOwnPropertyDescriptor(prototype, attr);

      if (originalDescriptor) {
        Object.defineProperty(element, attr, {
          get: function() { return originalDescriptor.get.call(this); },
          set: function(value) {
            // 跳过module脚本
            if (lowerTag === 'script' && (this.type === 'module' || this.getAttribute('type') === 'module')) {
              originalDescriptor.set.call(this, value);
              return;
            }
            const rewritten = rewriteURL(value);
            originalDescriptor.set.call(this, rewritten);
          }
        });
      }
    }

    return element;
  };

  // ==================== 拦截 fetch/XHR/importScripts ====================
  if (window.fetch) {
    const originalFetch = window.fetch;
    window.fetch = function(input, init) {
      if (typeof input === 'string') input = rewriteURL(input);
      if (input instanceof Request) {
        const newUrl = rewriteURL(input.url);
        input = new Request(newUrl, input);
      }
      return originalFetch.call(this, input, init);
    };
  }

  if (window.XMLHttpRequest) {
    const originalXHROpen = XMLHttpRequest.prototype.open;
    XMLHttpRequest.prototype.open = function(method, url, ...args) {
      return originalXHROpen.call(this, method, rewriteURL(url), ...args);
    };
  }

  if (window.importScripts) {
    const originalImportScripts = window.importScripts;
    window.importScripts = function(...urls) {
      return originalImportScripts.apply(this, urls.map(rewriteURL));
    };
  }

  // ==================== 拦截 静态HTML标签（核心：修复静态资源生效） ====================
  const observer = new MutationObserver(mutations => {
    mutations.forEach(mutation => {
      mutation.addedNodes.forEach(node => {
        if (node.nodeType !== 1) return;
        const tag = node.tagName;

        // 全覆盖静态资源标签
        const rewriteMap = {
          SCRIPT: 'src',
          LINK: 'href',
          IMG: 'src',
          IFRAME: 'src',
          VIDEO: 'src',
          AUDIO: 'src'
        };

        const attr = rewriteMap[tag];
        if (attr && node[attr]) {
          const original = node[attr];
          const rewritten = rewriteURL(original);
          if (original !== rewritten) node[attr] = rewritten;
        }
      });
    });
  });

  // 监听整个文档
  observer.observe(document.documentElement, { childList: true, subtree: true });

  // ==================== 拦截 History/WebSocket/EventSource ====================
  if (window.history) {
    const originalPushState = history.pushState;
    const originalReplaceState = history.replaceState;

    history.pushState = function(s, t, url) {
      return originalPushState.call(this, s, t, url ? rewriteURL(url) : url);
    };
    history.replaceState = function(s, t, url) {
      return originalReplaceState.call(this, s, t, url ? rewriteURL(url) : url);
    };
  }

  if (window.WebSocket) {
    const Ws = window.WebSocket;
    window.WebSocket = function(u, p) { return new Ws(rewriteURL(u), p); };
    window.WebSocket.prototype = Ws.prototype;
  }

  if (window.EventSource) {
    const Es = window.EventSource;
    window.EventSource = function(u, c) { return new Es(rewriteURL(u), c); };
    window.EventSource.prototype = Es.prototype;
  }

  log('拦截脚本加载完成 ✅');
})();
`, proxyPrefix, token, debugStr)

	return script
}
