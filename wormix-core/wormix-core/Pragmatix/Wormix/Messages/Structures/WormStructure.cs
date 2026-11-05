using wormix_core.Extensions;
using wormix_core.Pragmatix.Wormix.Messages.Interfaces;

namespace wormix_core.Pragmatix.Wormix.Messages.Structures;

public struct WormStructure : ISerializable
{
    public uint OwnerId;
    public string SocialOwnerId;

    public uint Armor;
    public uint Attack;

    public uint Level;
    public uint Experience;

    public short Hat;

    /// <summary>
    /// Equipped artifact (item id 2001+), 0 = none. Not in the original 1.05.0 protocol: added for the
    /// artifacts port, read by the patched client right after SocialOwnerId.
    /// </summary>
    public short Artifact;

    public uint GetSize()
    {
        return
            (uint)(
                4 //OwnerId

                + 4 //Armor
                + 4 //Attack

                + 4 //Level
                + 4 //Experience
                + 2 //Hat
                + 2 + SocialOwnerId.Length
                + 2 //Artifact
            );
    }

    public void Serialize(Stream output)
    {
        BinaryWriter bw = new BinaryWriter(output);
        bw.WriteUInt32Be(OwnerId);
        
        bw.WriteUInt32Be(Armor);
        bw.WriteUInt32Be(Attack);
        
        bw.WriteUInt32Be(Level);
        bw.WriteUInt32Be(Experience);
        
        bw.WriteUInt16Be((ushort)Hat);
        
        bw.WriteUTF8(SocialOwnerId);

        bw.WriteUInt16Be((ushort)Artifact);
    }
}