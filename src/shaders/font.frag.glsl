#if __VERSION__ >= 130
#define COMPAT_VARYING in
#define COMPAT_ATTRIBUTE in
#define COMPAT_TEXTURE texture
#define COMPAT_FRAGCOLOR FragColor
#if __VERSION__ >= 450
layout(location = 0) out vec4 FragColor;
#else
out vec4 FragColor;
#endif
#else
#define COMPAT_VARYING varying
#define COMPAT_ATTRIBUTE attribute
#define COMPAT_TEXTURE texture2D
#define COMPAT_FRAGCOLOR gl_FragColor
#endif

layout(location = 0) COMPAT_VARYING vec2 fragTexCoord;

layout(push_constant, std430) uniform u {
	vec4 textColor;
};
layout(binding = 0) uniform sampler2D tex;

void main()
{
    vec4 sampled = vec4(1.0, 1.0, 1.0, COMPAT_TEXTURE(tex, fragTexCoord).r);
    COMPAT_FRAGCOLOR = min(textColor, vec4(1.0, 1.0, 1.0, 1.0)) * sampled;
}