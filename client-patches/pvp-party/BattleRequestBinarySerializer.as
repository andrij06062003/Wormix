package ru.pragmatix.wormix.serialization.client
{
   import flash.utils.ByteArray;
   import flash.utils.IDataInput;
   import flash.utils.IDataOutput;
   import ru.pragmatix.flox.serialization.interfaces.ICommandHeader;
   import ru.pragmatix.flox.serialization.interfaces.ICommandSerializer;
   import ru.pragmatix.flox.serialization.internals.BinaryCommandHeader;
   import ru.pragmatix.wormix.pvp.messages.client.BattleRequest;
   
   public class BattleRequestBinarySerializer implements ICommandSerializer
   {
      
      public function BattleRequestBinarySerializer()
      {
         super();
      }
      
      public function getCommandId() : uint
      {
         return 1002;
      }
      
      public function serializeCommand(obj:Object, stream:IDataOutput) : void
      {
         var self:BattleRequest = BattleRequest(obj);
         var header:BinaryCommandHeader = new BinaryCommandHeader();
         header.setCommandId(this.getCommandId());
         var byteArray:ByteArray = new ByteArray();
         byteArray.writeUTF(self.host);
         byteArray.writeInt(self.port);
         byteArray.writeUnsignedInt(self.mapId);
         byteArray.writeInt(self.seed);
         byteArray.writeUTF(self.enemyId);
         if(self.mode != 0)
         {
            byteArray.writeByte(self.mode);
         }
         header.setLength(byteArray.length);
         header.write(stream);
         stream.writeBytes(byteArray);
      }
      
      public function deserializeCommand(stream:IDataInput, header:ICommandHeader) : Object
      {
         var self:BattleRequest = new BattleRequest();
         self.host = stream.readUTF();
         self.port = stream.readInt();
         self.mapId = stream.readUnsignedInt();
         self.seed = stream.readInt();
         self.enemyId = stream.readUTF();
         return self;
      }
   }
}

