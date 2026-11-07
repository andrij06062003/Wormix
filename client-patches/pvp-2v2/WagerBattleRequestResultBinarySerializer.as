package ru.pragmatix.wormix.serialization.wager
{
   import flash.utils.ByteArray;
   import flash.utils.IDataInput;
   import flash.utils.IDataOutput;
   import ru.pragmatix.flox.gui.controls.MessageBox;
   import ru.pragmatix.flox.secure.SecuritySettings;
   import ru.pragmatix.flox.secure.SerializeSecurityUtils;
   import ru.pragmatix.flox.serialization.BinarySerializer;
   import ru.pragmatix.flox.serialization.interfaces.ICommandHeader;
   import ru.pragmatix.flox.serialization.interfaces.ICommandSerializer;
   import ru.pragmatix.wormix.gui.Messages;
   import ru.pragmatix.wormix.messages.structures.UserProfileStructure;
   import ru.pragmatix.wormix.net.MainConnection;
   import ru.pragmatix.wormix.pvp.messages.wager.WagerBattleRequestResult;
   
   public class WagerBattleRequestResultBinarySerializer implements ICommandSerializer
   {
      
      public function WagerBattleRequestResultBinarySerializer()
      {
         super();
      }
      
      public function getCommandId() : uint
      {
         return 2006;
      }
      
      public function serializeCommand(obj:Object, stream:IDataOutput) : void
      {
      }
      
      public function deserializeCommand(dataInput:IDataInput, header:ICommandHeader) : Object
      {
         var stream:ByteArray = new ByteArray();
         dataInput.readBytes(stream,0,dataInput.bytesAvailable);
         var self:WagerBattleRequestResult = new WagerBattleRequestResult();
         self.host = stream.readUTF();
         self.port = stream.readInt();
         self.mapId = stream.readUnsignedInt();
         stream.readShort();
         self.enemyProfile = UserProfileStructure(BinarySerializer.getStructureSerializer(UserProfileStructure).deserializeStructure(stream,null));
         stream.readShort();
         self.userProfile = UserProfileStructure(BinarySerializer.getStructureSerializer(UserProfileStructure).deserializeStructure(stream,null));
         self.firstPlayerId = stream.readUnsignedInt();
         self.seed = stream.readInt();
         self.battleId = stream.readUnsignedInt();
         var reagentSize:int = int(stream.readUnsignedShort());
         self.reagentsForBattle = new Vector.<int>();
         for(var i:int = 0; i < reagentSize; i++)
         {
            self.reagentsForBattle[i] = stream.readByte();
         }
         if(stream.bytesAvailable > 16)
         {
            var players:int = int(stream.readUnsignedByte());
            for(var j:int = 0; j < players; j++)
            {
               self.teamIds.push(int(stream.readUnsignedByte()));
               stream.readShort();
               self.teamProfiles.push(UserProfileStructure(BinarySerializer.getStructureSerializer(UserProfileStructure).deserializeStructure(stream,null)));
            }
         }
         var isSecure:Boolean = SerializeSecurityUtils.isSecure(stream,SecuritySettings.SECRET_KEY);
         if(!isSecure)
         {
            MainConnection.INSTANCE.close();
            MessageBox.showMsg(Messages.TITLE_ATTENTION,Messages.PROTOCOL_CHEATER,MessageBox.BTN_NONE);
            return null;
         }
         return self;
      }
   }
}

