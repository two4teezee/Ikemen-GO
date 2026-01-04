#if __VERSION__ >= 450
    #define COMPAT_TEXTURE texture
    layout(push_constant, std430) uniform u {
        layout(offset = 8) float CurrentTime; // Note: removed redundant 'uniform' keyword inside block
    };
    layout(binding = 0) uniform sampler2D Texture;
    layout(location = 0) in vec2 texcoord;
    layout(location = 0) out vec4 FragColor;
#else
    #if __VERSION__ >= 130 || defined(GL_ES)
        #define COMPAT_VARYING in
        #define COMPAT_TEXTURE texture
        #ifdef GL_ES
            precision highp float;
        #endif
        out vec4 FragColor;
    #else
        #define COMPAT_VARYING varying
        #define FragColor gl_FragColor
        #define COMPAT_TEXTURE texture2D
    #endif

    uniform sampler2D Texture;
    uniform float CurrentTime; // Keep it here for compatibility even if unused in main
    COMPAT_VARYING vec2 texcoord;
#endif

void main(void) {
    FragColor = COMPAT_TEXTURE(Texture, texcoord);
}