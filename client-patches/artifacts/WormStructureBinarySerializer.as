package ru.pragmatix.wormix.serialization.structures
{
   import flash.utils.IDataInput;
   import flash.utils.IDataOutput;
   import ru.pragmatix.flox.serialization.interfaces.IStructureSerializer;
   import ru.pragmatix.wormix.messages.structures.WormStructure;
   
   public class WormStructureBinarySerializer implements IStructureSerializer
   {
      
      public function WormStructureBinarySerializer()
      {
         super();
      }
      
      public function serializeStructure(objSelf:Object, stream:IDataOutput, extractedFields:Object) : void
      {
         var self:WormStructure = WormStructure(objSelf);
         stream.writeUnsignedInt(self.ownerId);
         stream.writeInt(self.armor);
         stream.writeInt(self.attack);
         stream.writeInt(self.level);
         stream.writeInt(self.experience);
         stream.writeShort(self.hat);
         stream.writeUTF(self.socialOwnerId);
         stream.writeShort(self.artifact);
      }
      
      public function deserializeStructure(stream:IDataInput, extractedFieldMap:Object) : Object
      {
         var self:WormStructure = new WormStructure();
         self.ownerId = stream.readUnsignedInt();
         self.armor = stream.readInt();
         self.attack = stream.readInt();
         self.level = stream.readInt();
         self.experience = stream.readInt();
         self.hat = stream.readShort();
         var loc:String = stream.readUTF();
         if(!loc)
         {
            self.socialOwnerId = self.ownerId.toString();
         }
         else
         {
            self.socialOwnerId = loc;
         }
         self.artifact = stream.readShort();
         return self;
      }
   }
}

