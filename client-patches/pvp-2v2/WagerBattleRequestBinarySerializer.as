package ru.pragmatix.wormix.serialization.wager
{
   import flash.utils.ByteArray;
   import flash.utils.IDataInput;
   import flash.utils.IDataOutput;
   import ru.pragmatix.flox.gui.controls.MessageBox;
   import ru.pragmatix.flox.secure.SecuritySettings;
   import ru.pragmatix.flox.secure.SerializeSecurityUtils;
   import ru.pragmatix.flox.serialization.interfaces.ICommandHeader;
   import ru.pragmatix.flox.serialization.interfaces.ICommandSerializer;
   import ru.pragmatix.flox.serialization.internals.BinaryCommandHeader;
   import ru.pragmatix.wormix.gui.Messages;
   import ru.pragmatix.wormix.net.MainConnection;
   import ru.pragmatix.wormix.pvp.messages.wager.WagerBattleRequest;
   
   public class WagerBattleRequestBinarySerializer implements ICommandSerializer
   {
      
      public function WagerBattleRequestBinarySerializer()
      {
         super();
      }
      
      public function getCommandId() : uint
      {
         return 2005;
      }
      
      public function serializeCommand(obj:Object, stream:IDataOutput) : void
      {
         var self:WagerBattleRequest = WagerBattleRequest(obj);
         var header:BinaryCommandHeader = new BinaryCommandHeader();
         header.setCommandId(this.getCommandId());
         var byteArray:ByteArray = new ByteArray();
         byteArray.writeUTF(self.host);
         byteArray.writeInt(self.port);
         byteArray.writeInt(self.wager);
         byteArray.writeUnsignedInt(self.mapId);
         if(self.mode != 0)
         {
            byteArray.writeByte(self.mode);
         }
         SerializeSecurityUtils.secure(byteArray,SecuritySettings.SECRET_KEY);
         header.setLength(byteArray.length);
         header.write(stream);
         stream.writeBytes(byteArray);
      }
      
      public function deserializeCommand(dataInput:IDataInput, header:ICommandHeader) : Object
      {
         var stream:ByteArray = new ByteArray();
         dataInput.readBytes(stream,0,dataInput.bytesAvailable);
         var self:WagerBattleRequest = new WagerBattleRequest();
         self.host = stream.readUTF();
         self.port = stream.readInt();
         self.wager = stream.readInt();
         self.mapId = stream.readUnsignedInt();
         var isSecure:Boolean = SerializeSecurityUtils.isSecure(stream,SecuritySettings.SECRET_KEY);
         if(!isSecure)
         {
            MainConnection.INSTANCE.close();
            MessageBox.showMsg(Messages.TITLE_ATTENTION,Messages.PROTOCOL_CHEATER,MessageBox.BTN_NONE);
         }
         return self;
      }
   }
}

