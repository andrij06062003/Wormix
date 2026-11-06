package ru.pragmatix.wormix.serialization.server
{
   import flash.utils.ByteArray;
   import flash.utils.IDataInput;
   import flash.utils.IDataOutput;
   import ru.pragmatix.flox.serialization.BinarySerializer;
   import ru.pragmatix.flox.serialization.interfaces.ICommandHeader;
   import ru.pragmatix.flox.serialization.interfaces.ICommandSerializer;
   import ru.pragmatix.flox.serialization.internals.BinaryCommandHeader;
   import ru.pragmatix.wormix.messages.structures.UserProfileStructure;
   import ru.pragmatix.wormix.pvp.messages.server.BattleOffer;
   
   public class BattleOfferBinarySerializer implements ICommandSerializer
   {
      
      public function BattleOfferBinarySerializer()
      {
         super();
      }
      
      public function getCommandId() : uint
      {
         return 2002;
      }
      
      public function serializeCommand(obj:Object, stream:IDataOutput) : void
      {
         var self:BattleOffer = BattleOffer(obj);
         var header:BinaryCommandHeader = new BinaryCommandHeader();
         header.setCommandId(this.getCommandId());
         var byteArray:ByteArray = new ByteArray();
         byteArray.writeUTF(self.host);
         byteArray.writeInt(self.port);
         byteArray.writeUnsignedInt(self.mapId);
         byteArray.writeInt(self.seed);
         var structureBoby:ByteArray = new ByteArray();
         BinarySerializer.getStructureSerializer(UserProfileStructure).serializeStructure(self.enemyProfile,structureBoby,null);
         byteArray.writeShort(structureBoby.length);
         byteArray.writeBytes(structureBoby);
         structureBoby = new ByteArray();
         BinarySerializer.getStructureSerializer(UserProfileStructure).serializeStructure(self.myProfile,structureBoby,null);
         byteArray.writeShort(structureBoby.length);
         byteArray.writeBytes(structureBoby);
         byteArray.writeUnsignedInt(self.battleId);
         header.setLength(byteArray.length);
         header.write(stream);
         stream.writeBytes(byteArray);
      }
      
      public function deserializeCommand(stream:IDataInput, header:ICommandHeader) : Object
      {
         var self:BattleOffer = new BattleOffer();
         self.host = stream.readUTF();
         self.port = stream.readInt();
         self.mapId = stream.readUnsignedInt();
         self.seed = stream.readInt();
         stream.readShort();
         self.enemyProfile = UserProfileStructure(BinarySerializer.getStructureSerializer(UserProfileStructure).deserializeStructure(stream,null));
         stream.readShort();
         self.myProfile = UserProfileStructure(BinarySerializer.getStructureSerializer(UserProfileStructure).deserializeStructure(stream,null));
         self.battleId = stream.readUnsignedInt();
         if(stream.bytesAvailable > 0)
         {
            self.kind = stream.readUnsignedByte();
         }
         return self;
      }
   }
}

