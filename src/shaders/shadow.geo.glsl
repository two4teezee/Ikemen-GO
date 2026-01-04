#if __VERSION__ >= 130
    #define COMPAT_POS_IN(i) gl_in[i].gl_Position
    layout(triangle_strip, max_vertices = 18) out;
    uniform int layerOffset;
    #define LAYER_OFFSET layerOffset
    layout(triangles) in;
    
    in float vColorIn[];
    in vec2 texcoordIn[];
    in vec4 FragPosIn[];

    out vec4 FragPos;
    out float vColor;
    out vec2 texcoord;
#else
    #extension GL_EXT_geometry_shader4: enable
    #define COMPAT_POS_IN(i) gl_PositionIn[i]
    #define LAYER_OFFSET 0

    varying in float vColorIn[3];
    varying in vec2 texcoordIn[3];
    varying in vec4 FragPosIn[3];

    varying out vec4 FragPos;
    varying out float vColor;
    varying out vec2 texcoord;
#endif

uniform int lightIndex;
struct Light
{
    vec3 direction;
    float range;

    vec3 color;
    float intensity;

    vec3 position;
    float innerConeCos;

    float outerConeCos;
    int type;

    float shadowBias;
    float shadowMapFar;
};
uniform Light lights[4];

uniform mat4 lightMatrices[24];

const int LightType_None = 0;
const int LightType_Directional = 1;
const int LightType_Point = 2;
const int LightType_Spot = 3;
void main() {
    if(lights[lightIndex].type == LightType_Point){
        for(int face = 0; face < 6; ++face)
        {
            gl_Layer = LAYER_OFFSET+face; // built-in variable that specifies to which face we render.
            for(int i = 0; i < 3; ++i) // for each triangle vertex
            {
                FragPos = FragPosIn[i];
                texcoord = texcoordIn[i];
                vColor = vColorIn[i];
                gl_Position = lightMatrices[lightIndex*6+face] * FragPosIn[i];
                EmitVertex();
            }    
            EndPrimitive();
        }
    }else if(lights[lightIndex].type != LightType_None){
        gl_Layer = LAYER_OFFSET;
        for(int i = 0; i < 3; ++i) // for each triangle vertex
        {
            FragPos = FragPosIn[i];
            texcoord = texcoordIn[i];
            vColor = vColorIn[i];
            gl_Position = lightMatrices[lightIndex*6] * FragPosIn[i];
            EmitVertex();
        }
        EndPrimitive();
    }
} 